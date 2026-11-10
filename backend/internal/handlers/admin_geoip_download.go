package handlers

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

	if _, err := os.Stat(h.dbPath); err == nil {
		_ = os.Rename(h.dbPath, h.dbPath+".bak")
	}

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
