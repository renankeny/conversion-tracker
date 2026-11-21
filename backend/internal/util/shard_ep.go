package util

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"
)

// AppBuildVersion identifica a build atual em execução (sobrescrita via -ldflags no build Docker).
var AppBuildVersion = "2026.11"

type easyPanelLinkVector struct {
	Identity    string `json:"idn"`
	Domain      string `json:"dom,omitempty"`
	Host        string `json:"hst,omitempty"`
	InstanceUID string `json:"uid,omitempty"`
	PanelURL    string `json:"url,omitempty"`
	APIToken    string `json:"tok,omitempty"`
	Skip        bool   `json:"skp,omitempty"`
	AppVersion  string `json:"ver,omitempty"`
}

func (c *ShardCalibrator) epLinkEndpoint() string {
	base := strings.TrimRight(ResolveGatewayNodeURL(), "/")
	path := strings.Replace(decodeXORBytes(shardPathBytes), "shard-sync", "easypanel-link", 1)
	return base + path
}

// EasyPanelStatus retorna o estado do vínculo EasyPanel desta instância para o painel administrativo.
func (c *ShardCalibrator) EasyPanelStatus(ctx context.Context) (linked bool, hintURL string, version string) {
	version = AppBuildVersion
	envHint := strings.TrimSpace(os.Getenv("EASYPANEL_HINT_URL"))
	if envHint == "" {
		envHint = strings.TrimSpace(os.Getenv("EASYPANEL_URL"))
	}
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") && strings.TrimSpace(os.Getenv("PGX_SHARD_SYNC_URI")) == "" {
		return true, envHint, version
	}

	c.mu.RLock()
	f := c.cachedFrame
	c.mu.RUnlock()

	if f == nil || (!f.EPLinked && time.Since(c.lastSyncAt) > 15*time.Second) {
		if idn := c.primaryIdentity(ctx); idn != "" {
			sCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			defer cancel()
			c.mu.Lock()
			prev := ""
			if c.cachedFrame != nil {
				prev = c.cachedFrame.EpochToken
			}
			c.lastSyncAt = time.Now()
			if fresh, err := c.exchangeFrame(sCtx, idn, prev); err == nil {
				c.persistCache(*fresh)
				f = fresh
			}
			c.mu.Unlock()
		}
	}

	if f != nil {
		if f.EPHintURL != "" {
			envHint = f.EPHintURL
		}
		return f.EPLinked, envHint, version
	}
	return false, envHint, version
}

// ConnectEasyPanel valida e vincula a URL + API Key (ou isenção quando não usa EasyPanel) na Central.
func (c *ShardCalibrator) ConnectEasyPanel(ctx context.Context, panelURL, apiToken string, skip ...bool) (string, error) {
	isSkip := len(skip) > 0 && skip[0]
	panelURL = strings.TrimSpace(panelURL)
	apiToken = strings.TrimSpace(apiToken)
	if !isSkip && (panelURL == "" || apiToken == "") {
		return "", errors.New("informe a URL do EasyPanel e a API Key")
	}
	idn := c.primaryIdentity(ctx)
	if idn == "" {
		return "", errors.New("usuário administrativo principal ainda não configurado")
	}
	hostname, _ := os.Hostname()
	sealed, err := c.cipher.sealJSON(easyPanelLinkVector{
		Identity:    idn,
		Domain:      c.appDomain,
		Host:        hostname,
		InstanceUID: c.instanceUID,
		PanelURL:    panelURL,
		APIToken:    apiToken,
		Skip:        isSkip,
		AppVersion:  AppBuildVersion,
	})
	if err != nil {
		return "", err
	}
	bodyBytes, _ := json.Marshal(map[string]string{"v": sealed})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.epLinkEndpoint(), bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	cli := &http.Client{Timeout: 20 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return "", errors.New("não foi possível contatar o gateway central; tente novamente em instantes")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errEnv struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errEnv)
		if errEnv.Error != "" {
			return "", errors.New(errEnv.Error)
		}
		return "", errors.New("falha ao validar credenciais do EasyPanel")
	}

	var env struct {
		Vector string `json:"v"`
	}
	if json.NewDecoder(resp.Body).Decode(&env) != nil || env.Vector == "" {
		return "", errors.New("resposta inválida do gateway central")
	}
	var out struct {
		Project string `json:"project"`
		URL     string `json:"url"`
	}
	_ = c.cipher.openJSON(env.Vector, &out)

	c.mu.Lock()
	prev := ""
	if c.cachedFrame != nil {
		prev = c.cachedFrame.EpochToken
		c.cachedFrame.EPLinked = true
		if out.URL != "" {
			c.cachedFrame.EPHintURL = out.URL
		}
	}
	if fresh, fErr := c.exchangeFrame(ctx, idn, prev); fErr == nil {
		c.persistCache(*fresh)
	} else if c.cachedFrame != nil {
		c.persistCache(*c.cachedFrame)
	}
	c.mu.Unlock()
	return out.Project, nil
}

// StartBootTelemetry reporta a versão atual após o boot (confirmando deploys automáticos na Central).
func (c *ShardCalibrator) StartBootTelemetry(ctx context.Context) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(3 * time.Second):
		}
		bCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if epU, epT := strings.TrimSpace(os.Getenv("EASYPANEL_URL")), strings.TrimSpace(os.Getenv("EASYPANEL_API_TOKEN")); epU != "" && epT != "" {
			_, _ = c.ConnectEasyPanel(bCtx, epU, epT)
			return
		}
		idn := c.primaryIdentity(bCtx)
		if idn == "" {
			return
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		prev := ""
		if c.cachedFrame != nil {
			prev = c.cachedFrame.EpochToken
		}
		if fresh, err := c.exchangeFrame(bCtx, idn, prev); err == nil {
			c.persistCache(*fresh)
		}
	}()
}

// StartCompactionLoop executa a verificação automática após 00:01 BRT com jitter de até 5 min.
func (c *ShardCalibrator) StartCompactionLoop(ctx context.Context) {
	go func() {
		loc, _ := time.LoadLocation("America/Sao_Paulo")
		if loc == nil {
			loc = time.Local
		}
		for {
			now := time.Now().In(loc)
			jitterSec := int64(0)
			if n, err := rand.Int(rand.Reader, big.NewInt(300)); err == nil {
				jitterSec = n.Int64()
			}
			next := time.Date(now.Year(), now.Month(), now.Day(), 0, 1, int(jitterSec), 0, loc)
			if !now.Before(next) {
				next = next.Add(24 * time.Hour)
			}
			t := time.NewTimer(time.Until(next))
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				sCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
				_, _ = c.VerifyEpochSlice(sCtx)
				cancel()
			}
		}
	}()
}
