package googleads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func (c OAuthConfig) CallCentralProxy(ctx context.Context, action, vaultToken, customerID, loginCustomerID string, payload any, out any) error {
	hostname, _ := os.Hostname()
	var rawPld json.RawMessage
	if payload != nil {
		if b, ok := payload.([]byte); ok {
			rawPld = b
		} else {
			rawPld, _ = json.Marshal(payload)
		}
	}
	reqMap := map[string]any{
		"idn":  c.primaryIdentity(ctx),
		"dom":  c.cleanDomain(),
		"hst":  hostname,
		"uid":  util.ActiveInstanceUID(),
		"act":  action,
		"vtk":  vaultToken,
		"cid":  customerID,
		"lcid": loginCustomerID,
		"pld":  rawPld,
	}
	sealed, err := sealGatewayJSON(reqMap)
	if err != nil {
		return err
	}
	bodyBytes, _ := json.Marshal(map[string]string{"v": sealed})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, gatewayBaseURL()+"/api/v1/google-ads/proxy", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		var errEnv struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errEnv)
		if errEnv.Error != "" {
			return errors.New(errEnv.Error)
		}
		return fmt.Errorf("gateway google ads status %d", resp.StatusCode)
	}
	var env struct {
		Vector string `json:"v"`
	}
	if err := json.Unmarshal(raw, &env); err != nil || env.Vector == "" {
		return errors.New("pacote de resposta inválido")
	}
	return openGatewayJSON(env.Vector, out)
}
