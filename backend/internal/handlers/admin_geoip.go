package handlers

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/renankeny/conversion-tracker/backend/internal/geoip"
)

const (
	geoipEditionID = "GeoLite2-City"
	geoipDownload  = "https://download.maxmind.com/app/geoip_download"
)

// AdminGeoIP atualiza a base GeoLite2-City baixando da MaxMind via license key.
// Endpoint público apenas pra admins autenticados.
type AdminGeoIP struct {
	licenseKey string
	dbPath     string
	lookup     *geoip.Lookup
	refreshing atomic.Bool
}

func NewAdminGeoIP(licenseKey, dbPath string, lookup *geoip.Lookup) *AdminGeoIP {
	return &AdminGeoIP{
		licenseKey: licenseKey,
		dbPath:     dbPath,
		lookup:     lookup,
	}
}

// Status retorna informações sobre a base atualmente carregada
// (path, exists, size, modified). Útil pro painel mostrar estado.
func (h *AdminGeoIP) Status(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{
		"path":            h.dbPath,
		"license_key_set": h.licenseKey != "",
		"loaded":          h.lookup != nil && h.lookup.Available(),
	}
	if h.dbPath != "" {
		if info, err := os.Stat(h.dbPath); err == nil {
			out["exists"] = true
			out["size_bytes"] = info.Size()
			out["modified_at"] = info.ModTime().Format(time.RFC3339)
		} else {
			out["exists"] = false
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// Refresh baixa a base GeoLite2-City da MaxMind, valida SHA256, extrai o
// .mmdb e substitui o arquivo em h.dbPath. Recarrega o reader em memória.
//
// É síncrono e pode demorar 30-90s. Use timeouts no proxy se necessário.
func (h *AdminGeoIP) Refresh(w http.ResponseWriter, r *http.Request) {
	if h.licenseKey == "" {
		writeError(w, http.StatusBadRequest, "MAXMIND_LICENSE_KEY não configurada no servidor")
		return
	}
	if h.dbPath == "" {
		writeError(w, http.StatusBadRequest, "GEOIP_DB_PATH não configurado no servidor")
		return
	}

	// Evita refresh concorrente.
	if !h.refreshing.CompareAndSwap(false, true) {
		writeError(w, http.StatusConflict, "uma atualização já está em andamento")
		return
	}
	defer h.refreshing.Store(false)

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	result, err := h.downloadAndInstall(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao atualizar GeoIP: "+err.Error())
		return
	}

	// Recarrega o reader em memória.
	if h.lookup != nil {
		if reloadErr := h.lookup.Reload(h.dbPath); reloadErr != nil {
			result["reload_error"] = reloadErr.Error()
		} else {
			result["reloaded"] = true
		}
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *AdminGeoIP) downloadAndInstall(ctx context.Context) (map[string]any, error) {
	tmp, err := os.MkdirTemp("", "geoip-*")
	if err != nil {
		return nil, fmt.Errorf("mkdir temp: %w", err)
	}
	defer os.RemoveAll(tmp)

	archivePath := filepath.Join(tmp, "geoip.tar.gz")

	// 1. Download archive
	archiveURL := fmt.Sprintf("%s?edition_id=%s&license_key=%s&suffix=tar.gz",
		geoipDownload, geoipEditionID, url.QueryEscape(h.licenseKey))
	if err := download(ctx, archiveURL, archivePath); err != nil {
		return nil, fmt.Errorf("download archive: %w", err)
	}

	// 2. Download checksum
	shaPath := archivePath + ".sha256"
	if err := download(ctx, archiveURL+".sha256", shaPath); err != nil {
		return nil, fmt.Errorf("download checksum: %w", err)
	}

	expected, err := readChecksum(shaPath)
	if err != nil {
		return nil, fmt.Errorf("checksum file: %w", err)
	}

	// 3. Validar SHA256
	actual, err := sha256File(archivePath)
	if err != nil {
		return nil, fmt.Errorf("compute sha256: %w", err)
	}
	if !strings.EqualFold(actual, expected) {
		return nil, fmt.Errorf("checksum mismatch: expected %s got %s", expected, actual)
	}

	// 4. Extrair .mmdb do tar.gz
	mmdbPath, err := extractMMDB(archivePath, tmp)
	if err != nil {
		return nil, fmt.Errorf("extract: %w", err)
	}

	// 5. Mover atomicamente pra h.dbPath
	if err := os.MkdirAll(filepath.Dir(h.dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir dest: %w", err)
	}

	// Backup do anterior, se existir.
	if _, err := os.Stat(h.dbPath); err == nil {
		_ = os.Rename(h.dbPath, h.dbPath+".bak")
	}

	// Tenta rename (mesmo FS); se falhar, faz copy + remove.
	if err := os.Rename(mmdbPath, h.dbPath); err != nil {
		if copyErr := copyFile(mmdbPath, h.dbPath); copyErr != nil {
			return nil, fmt.Errorf("install file: %w (rename: %v)", copyErr, err)
		}
	}

	info, _ := os.Stat(h.dbPath)
	out := map[string]any{
		"path":      h.dbPath,
		"installed": true,
	}
	if info != nil {
		out["size_bytes"] = info.Size()
		out["modified_at"] = info.ModTime().Format(time.RFC3339)
	}
	return out, nil
}

func download(ctx context.Context, src, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func readChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return "", errors.New("checksum vazio")
	}
	return parts[0], nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extractMMDB encontra e extrai o primeiro arquivo .mmdb do tar.gz.
func extractMMDB(archive, dst string) (string, error) {
	f, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if !strings.HasSuffix(hdr.Name, ".mmdb") {
			continue
		}
		outPath := filepath.Join(dst, filepath.Base(hdr.Name))
		out, err := os.Create(outPath)
		if err != nil {
			return "", err
		}
		// Limita a 256MB pra evitar tarbomb.
		if _, err := io.CopyN(out, tr, 256<<20); err != nil && err != io.EOF {
			out.Close()
			return "", err
		}
		out.Close()
		return outPath, nil
	}
	return "", errors.New("nenhum arquivo .mmdb encontrado no archive")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// init garante que o linter não reclame caso json não seja usado em algum branch.
var _ = json.Marshal
