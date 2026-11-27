package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
	"github.com/renankeny/conversion-tracker/backend/internal/evolution"
	"github.com/renankeny/conversion-tracker/backend/internal/queue"
	"github.com/renankeny/conversion-tracker/backend/internal/repo"
	"github.com/renankeny/conversion-tracker/backend/internal/uazapi"
)

var evolutionReconnectRetryWait = 2500 * time.Millisecond

func syncInstanceLiveStatus(ctx context.Context, r *repo.WhatsAppRepo, inst *domain.WhatsAppInstance, publicAPIURL string) error {
	if inst.IsWAHA() {
		return syncWAHALiveStatus(ctx, r, inst)
	}
	if inst.IsExternalUazapi() {
		return syncUazapiLiveStatus(ctx, r, inst, publicAPIURL)
	}
	return syncEvolutionLiveStatus(ctx, r, inst)
}

func syncUazapiLiveStatus(ctx context.Context, r *repo.WhatsAppRepo, inst *domain.WhatsAppInstance, publicAPIURL string) error {
	if inst.ExternalAPIURL == "" || inst.InstanceToken == "" {
		return nil
	}
	uazCli := uazapi.New(inst.ExternalAPIURL, inst.InstanceToken)
	if publicAPIURL != "" && inst.WebhookSecret != "" {
		whURL := buildProviderWebhookURL(publicAPIURL, inst.Provider, inst.WebhookSecret)
		_ = uazCli.ConfigureWebhook(ctx, whURL, inst.WebhookSecret)
	}
	st, err := uazCli.GetStatus(ctx)
	if err != nil {
		return err
	}
	if st.Connected {
		ownerBare := queue.NormalizeBareJID(st.OwnerJID)
		if ownerBare == "" {
			ownerBare = inst.OwnerJID
		}
		profName := st.ProfileName
		if profName == "" {
			profName = inst.ProfileName
		}
		profPic := st.ProfilePicURL
		if profPic == "" {
			profPic = inst.ProfilePicURL
		}
		if r != nil {
			_ = r.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, profPic)
		}
		inst.ConnectionStatus = "OPEN"
		inst.OwnerJID = ownerBare
		inst.ProfileName = profName
		inst.ProfilePicURL = profPic
		return nil
	}
	status := "CLOSED"
	if st.Status == "connecting" || st.Qrcode != "" {
		status = "CONNECTING"
	}
	if r != nil {
		_ = r.ResetInstanceSession(ctx, inst.ID, status)
	}
	inst.ConnectionStatus = status
	inst.OwnerJID = ""
	inst.ProfileName = ""
	inst.ProfilePicURL = ""
	return nil
}

// syncEvolutionLiveStatus consulta o status na Evolution Go respeitando a janela de 2s do ReconnectClient.
func syncEvolutionLiveStatus(ctx context.Context, r *repo.WhatsAppRepo, inst *domain.WhatsAppInstance) error {
	cli := evolution.NewClient(inst.ServerURL, inst.ServerAPIKey)
	st, err := cli.GetStatus(ctx, inst.InstanceToken)
	if err != nil {
		return err
	}

	target := inst.EvolutionUUID
	if target == "" {
		target = inst.InstanceName
	}
	info, infoErr := cli.GetInfo(ctx, target)

	infoHasJID := info != nil && strings.TrimSpace(info.JID) != ""
	hasSavedPairing := st.LoggedIn || infoHasJID || strings.TrimSpace(inst.OwnerJID) != ""
	isConnected := st.Connected && (st.LoggedIn || (info != nil && info.Connected))

	// Durante ReconnectClient, a Evolution Go remove o client em memória por 2s antes de recriar a conexão.
	if !isConnected && hasSavedPairing {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(evolutionReconnectRetryWait):
		}
		if st2, err2 := cli.GetStatus(ctx, inst.InstanceToken); err2 == nil && st2 != nil {
			st = st2
		}
		if info2, ierr2 := cli.GetInfo(ctx, target); ierr2 == nil && info2 != nil {
			info = info2
			infoErr = nil
			infoHasJID = strings.TrimSpace(info.JID) != ""
		}
		isConnected = st.Connected && (st.LoggedIn || (info != nil && info.Connected))
	}

	if isConnected {
		ownerBare := queue.NormalizeBareJID(st.MyJID)
		if ownerBare == "" && infoHasJID {
			ownerBare = queue.NormalizeBareJID(info.JID)
		}
		if ownerBare == "" {
			ownerBare = inst.OwnerJID
		}
		profName := strings.TrimSpace(st.Name)
		if info != nil && strings.TrimSpace(info.ProfileName) != "" {
			profName = strings.TrimSpace(info.ProfileName)
		}
		if profName == "" {
			profName = inst.ProfileName
		}
		profPic := ""
		if info != nil && strings.TrimSpace(info.ProfilePicURL) != "" {
			profPic = strings.TrimSpace(info.ProfilePicURL)
		}
		if profPic == "" && strings.TrimSpace(inst.ProfilePicURL) == "" && ownerBare != "" {
			profPic = cli.GetUserAvatar(ctx, inst.InstanceToken, ownerBare)
		}
		if profPic == "" {
			profPic = inst.ProfilePicURL
		}

		if r != nil {
			_ = r.UpdateInstanceOwnerAndProfile(ctx, inst.ID, "OPEN", ownerBare, profName, profPic)
		}
		inst.ConnectionStatus = "OPEN"
		inst.OwnerJID = ownerBare
		inst.ProfileName = profName
		inst.ProfilePicURL = profPic
		return nil
	}

	if st.Connected || (info != nil && strings.TrimSpace(info.Qrcode) != "") {
		if r != nil {
			_ = r.UpdateInstanceStatus(ctx, inst.ID, "CONNECTING")
		}
		inst.ConnectionStatus = "CONNECTING"
		return nil
	}

	// Se o aparelho continua pareado (JID salvo na Evolution Go), não limpamos a sessão como CLOSED.
	if st.LoggedIn || infoHasJID || (infoErr != nil && strings.TrimSpace(inst.OwnerJID) != "") {
		if r != nil {
			_ = r.UpdateInstanceStatus(ctx, inst.ID, "DISCONNECTED")
		}
		inst.ConnectionStatus = "DISCONNECTED"
		return nil
	}

	if r != nil {
		_ = r.ResetInstanceSession(ctx, inst.ID, "CLOSED")
	}
	inst.ConnectionStatus = "CLOSED"
	inst.OwnerJID = ""
	inst.ProfileName = ""
	inst.ProfilePicURL = ""
	return nil
}
