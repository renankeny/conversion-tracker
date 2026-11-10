package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

var nonDigitsRe = regexp.MustCompile(`\D+`)

func stripAuthPrefix(auth string) string {
	auth = strings.TrimSpace(auth)
	for _, prefix := range []string{"Bearer ", "bearer ", "Basic ", "basic ", "Token ", "token "} {
		if strings.HasPrefix(auth, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(auth, prefix))
		}
	}
	return auth
}

// readWebhookBody lê o corpo da requisição preservando r.Body caso outro trecho precise reler.
func readWebhookBody(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

// decodeWebhookPayload aceita tanto JSON quanto application/x-www-form-urlencoded
// (incluindo chaves estilo PHP como venda[status]=Completa&comprador[email]=x@y.com).
// Retorna os bytes JSON normalizados para uso com ExtractUserIDDeep.
func decodeWebhookPayload(r *http.Request, dst any) ([]byte, error) {
	raw, err := readWebhookBody(r)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(raw)
	ct := strings.ToLower(r.Header.Get("Content-Type"))

	isForm := strings.Contains(ct, "application/x-www-form-urlencoded") ||
		(len(trimmed) > 0 && trimmed[0] != '{' && trimmed[0] != '[' && bytes.Contains(trimmed, []byte("=")))

	jsonBytes := trimmed
	if isForm {
		converted, ferr := formBodyToJSON(string(trimmed))
		if ferr != nil {
			return raw, ferr
		}
		jsonBytes = converted
	}

	dec := json.NewDecoder(bytes.NewReader(jsonBytes))
	dec.UseNumber()
	if err := dec.Decode(dst); err != nil {
		return raw, err
	}
	return jsonBytes, nil
}

// formBodyToJSON converte query-string / form-urlencoded (ex: venda[codigo]=10&chave_unica=abc)
// em um objeto JSON aninhado.
func formBodyToJSON(formStr string) ([]byte, error) {
	vals, err := url.ParseQuery(formStr)
	if err != nil {
		return nil, err
	}
	root := make(map[string]any, len(vals))
	for key, list := range vals {
		if len(list) == 0 {
			continue
		}
		val := list[0]
		setBracketKey(root, key, val)
	}
	return json.Marshal(root)
}

func setBracketKey(root map[string]any, rawKey, val string) {
	if !strings.Contains(rawKey, "[") {
		root[rawKey] = val
		return
	}
	clean := strings.ReplaceAll(rawKey, "]", "")
	parts := strings.Split(clean, "[")
	curr := root
	for i := 0; i < len(parts)-1; i++ {
		p := parts[i]
		if p == "" {
			continue
		}
		next, ok := curr[p].(map[string]any)
		if !ok {
			next = make(map[string]any)
			curr[p] = next
		}
		curr = next
	}
	last := parts[len(parts)-1]
	if last != "" {
		curr[last] = val
	}
}

// joinPhoneDDD concatena DDI/DDD + telefone sem duplicar código de área.
func joinPhoneDDD(ddi, ddd, number string) string {
	num := nonDigitsRe.ReplaceAllString(number, "")
	area := nonDigitsRe.ReplaceAllString(ddd, "")
	country := nonDigitsRe.ReplaceAllString(ddi, "")
	if num == "" {
		return ""
	}
	if area != "" && len(num) <= 9 && !strings.HasPrefix(num, area) {
		num = area + num
	}
	if country != "" && len(num) <= 11 && !strings.HasPrefix(num, country) {
		num = country + num
	}
	return num
}

// anyString converte IDs que podem chegar como string, int ou float JSON em string limpa.
func anyString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case json.Number:
		return x.String()
	case float64:
		return util.FormatFloat(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func webhookPixelQuery(r *http.Request) string {
	return util.FirstNonEmpty(r.URL.Query().Get("pixel"), r.URL.Query().Get("token"))
}

