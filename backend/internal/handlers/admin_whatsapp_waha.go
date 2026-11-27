package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
	"github.com/renankeny/conversion-tracker/backend/internal/waha"
)

var wahaStartingRetryWait = 900 * time.Millisecond

func wahaClientForInstance(inst *domain.WhatsAppInstance) *waha.Client {
	return waha.NewClient(inst.ServerURL, inst.ServerAPIKey)
}

func provisionWAHAInstance(ctx context.Context, r *repo.WhatsAppRepo, slug, webhookURL string) (serverID int64, statusCode int, errMsg string) {
	server, err := r.GetGlobalServerByProvider(ctx, domain.WhatsAppProviderWAHA)
	if err != nil || server == nil {
		return 0, 400, "configure a URL e API Key da WAHA API antes de criar uma conexão WAHA"
	}
	if _, err := waha.NewClient(server.URL, server.APIKey).CreateAndStartSession(ctx, slug, webhookURL); err != nil {
		return 0, 502, "erro ao criar sessão na WAHA API: " + err.Error()
	}
	return server.ID, 0, ""
}

func rollbackWAHAInstance(ctx context.Context, r *repo.WhatsAppRepo, sessionName string) {
	if srv, _ := r.GetGlobalServerByProvider(ctx, domain.WhatsAppProviderWAHA); srv != nil {
		_ = waha.NewClient(srv.URL, srv.APIKey).DeleteSession(ctx, sessionName)
	}
}

func waitWAHASessionReadyForAuth(ctx context.Context, cli *waha.Client, sessionName string, cur *waha.SessionInfo) (*waha.SessionInfo, error) {
	info := cur
	for attempt := 0; attempt < 4; attempt++ {
		if info != nil {
			st := strings.ToUpper(strings.TrimSpace(info.Status))
			if st == "WORKING" || st == "SCAN_QR_CODE" || st == "FAILED" {
				return info, nil
			}
		}
		select {
		case <-ctx.Done():
			return info, ctx.Err()
		case <-time.After(wahaStartingRetryWait):
		}
		next, err := cli.GetSession(ctx, sessionName)
		if err != nil {
			return info, err
		}
		info = next
	}
	return info, nil
}

// fetchWAHAQRCode garante que a sessão esteja rodando na WAHA e retorna QR Code ou estado conectado.
func fetchWAHAQRCode(ctx context.Context, r *repo.WhatsAppRepo, inst *domain.WhatsAppInstance, publicAPIURL string) (qrDataURL string, connected bool, err error) {
	if inst.ServerURL == "" {
		return "", false, fmt.Errorf("servidor WAHA não configurado para esta instância")
	}
	cli := wahaClientForInstance(inst)
	whURL := buildProviderWebhookURL(publicAPIURL, domain.WhatsAppProviderWAHA, inst.WebhookSecret)

	info, err := cli.EnsureSessionRunning(ctx, inst.InstanceName, whURL)
	if err != nil {
		return "", false, err
	}
	info, _ = waitWAHASessionReadyForAuth(ctx, cli, inst.InstanceName, info)
	if info != nil && strings.EqualFold(strings.TrimSpace(info.Status), "WORKING") {
		_ = syncWAHALiveStatus(ctx, r, inst)
		return "", true, nil
	}

	qr, qerr := cli.GetQRCode(ctx, inst.InstanceName)
	if qerr != nil {
		if syncErr := syncWAHALiveStatus(ctx, r, inst); syncErr == nil && inst.ConnectionStatus == "OPEN" {
			return "", true, nil
		}
		return "", false, qerr
	}
	return qr, false, nil
}

func pairWAHAPhone(ctx context.Context, inst *domain.WhatsAppInstance, publicAPIURL, phone string) (string, error) {
	if inst.ServerURL == "" {
		return "", fmt.Errorf("servidor WAHA não configurado para esta instância")
	}
	cli := wahaClientForInstance(inst)
	whURL := buildProviderWebhookURL(publicAPIURL, domain.WhatsAppProviderWAHA, inst.WebhookSecret)
	info, err := cli.EnsureSessionRunning(ctx, inst.InstanceName, whURL)
	if err != nil {
		return "", err
	}
	_, _ = waitWAHASessionReadyForAuth(ctx, cli, inst.InstanceName, info)
	return cli.RequestPairingCode(ctx, inst.InstanceName, phone)
}

func syncWAHALiveStatus(ctx context.Context, r *repo.WhatsAppRepo, inst *domain.WhatsAppInstance) error {
	if inst.ServerURL == "" || inst.InstanceName == "" {
		return nil
	}
	cli := wahaClientForInstance(inst)
	sess, err := cli.GetSession(ctx, inst.InstanceName)
	if errors.Is(err, waha.ErrSessionNotFound) {
		if r != nil {
			_ = r.ResetInstanceSession(ctx, inst.ID, "CLOSED")
		}
		inst.ConnectionStatus = "CLOSED"
		inst.OwnerJID = ""
		inst.ProfileName = ""
		inst.ProfilePicURL = ""
		return nil
	}
	if err != nil {
		return err
	}

	status := strings.ToUpper(strings.TrimSpace(sess.Status))
	switch status {
	case "WORKING":
		ownerBare, profName, profPic := "", "", ""
		if sess.Me != nil {
			ownerBare = util.FirstNonEmpty(queue.NormalizeWAHAJID(sess.Me.JID), queue.NormalizeWAHAJID(sess.Me.ID))
			profName = strings.TrimSpace(sess.Me.PushName)
		}
		if prof, perr := cli.GetProfile(ctx, inst.InstanceName); perr == nil && prof != nil {
			ownerBare = util.FirstNonEmpty(ownerBare, queue.NormalizeWAHAJID(prof.ID))
			profName = util.FirstNonEmpty(profName, prof.Name)
			profPic = util.FirstNonEmpty(profPic, prof.Picture)
		}
		ownerBare = util.FirstNonEmpty(ownerBare, inst.OwnerJID)
		profName = util.FirstNonEmpty(profName, inst.ProfileName)
		profPic = util.FirstNonEmpty(profPic, inst.ProfilePicURL)
		if r != nil {
			_ = r.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, profPic)
		}
		inst.ConnectionStatus = "OPEN"
		inst.OwnerJID = ownerBare
		inst.ProfileName = profName
		inst.ProfilePicURL = profPic
		return nil

	case "STARTING", "SCAN_QR_CODE":
		if r != nil {
			_ = r.UpdateInstanceStatus(ctx, inst.ID, "CONNECTING")
		}
		inst.ConnectionStatus = "CONNECTING"
		return nil

	case "FAILED":
		next := "CLOSED"
		if strings.TrimSpace(inst.OwnerJID) != "" {
			next = "DISCONNECTED"
		}
		if r != nil {
			if next == "CLOSED" {
				_ = r.ResetInstanceSession(ctx, inst.ID, next)
			} else {
				_ = r.UpdateInstanceStatus(ctx, inst.ID, next)
			}
		}
		inst.ConnectionStatus = next
		return nil

	default:
		if r != nil {
			_ = r.ResetInstanceSession(ctx, inst.ID, "CLOSED")
		}
		inst.ConnectionStatus = "CLOSED"
		inst.OwnerJID = ""
		inst.ProfileName = ""
		inst.ProfilePicURL = ""
		return nil
	}
}
