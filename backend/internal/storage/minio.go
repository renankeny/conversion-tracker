package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"

	"github.com/renankeny/conversion-tracker/backend/internal/config"
)

type MediaStorage struct {
	client *minio.Client
	bucket string
}

func NewMediaStorage(cfg *config.Config) (*MediaStorage, error) {
	if !cfg.MinIOConfigured() {
		return nil, nil
	}
	cli, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
		Region: cfg.MinIORegion,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client: %w", err)
	}
	return &MediaStorage{
		client: cli,
		bucket: cfg.MinIOBucket,
	}, nil
}

func (s *MediaStorage) Enabled() bool {
	return s != nil && s.client != nil && s.bucket != ""
}

func (s *MediaStorage) Bucket() string {
	if s == nil {
		return ""
	}
	return s.bucket
}

// EnsureBucket garante que o bucket privado exista e aplica regra de Lifecycle
// (safety net com +1 dia além da retenção do banco para limpar eventuais órfãos).
func (s *MediaStorage) EnsureBucket(ctx context.Context, retentionDays int) error {
	if !s.Enabled() {
		return nil
	}
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check minio bucket %q: %w", s.bucket, err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create minio bucket %q: %w", s.bucket, err)
		}
		slog.Info("minio bucket created", "bucket", s.bucket)
	}

	if retentionDays > 0 {
		lcCfg := lifecycle.NewConfiguration()
		lcCfg.Rules = []lifecycle.Rule{
			{
				ID:     "whatsapp-media-safety-expiry",
				Status: "Enabled",
				Expiration: lifecycle.Expiration{
					Days: lifecycle.ExpirationDays(retentionDays + 1),
				},
			},
		}
		if err := s.client.SetBucketLifecycle(ctx, s.bucket, lcCfg); err != nil {
			slog.Warn("minio bucket lifecycle skipped (provider may not support it)", "bucket", s.bucket, "err", err)
		}
	}
	return nil
}

func (s *MediaStorage) PutMedia(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) (int64, error) {
	if !s.Enabled() {
		return 0, fmt.Errorf("minio storage not configured")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	info, err := s.client.PutObject(ctx, s.bucket, objectKey, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return 0, fmt.Errorf("put minio object %q: %w", objectKey, err)
	}
	return info.Size, nil
}

// OpenObject retorna o *minio.Object (implementa io.ReadSeeker para http.ServeContent + HTTP Range)
// junto com seus metadados. O chamador deve executar obj.Close().
func (s *MediaStorage) OpenObject(ctx context.Context, objectKey string) (*minio.Object, minio.ObjectInfo, error) {
	if !s.Enabled() {
		return nil, minio.ObjectInfo{}, fmt.Errorf("minio storage not configured")
	}
	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, minio.ObjectInfo{}, fmt.Errorf("get minio object %q: %w", objectKey, err)
	}
	stat, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, minio.ObjectInfo{}, fmt.Errorf("stat minio object %q: %w", objectKey, err)
	}
	return obj, stat, nil
}

func (s *MediaStorage) DeleteBatch(ctx context.Context, objectKeys []string) error {
	if !s.Enabled() || len(objectKeys) == 0 {
		return nil
	}
	objectsCh := make(chan minio.ObjectInfo, len(objectKeys))
	for _, k := range objectKeys {
		if k != "" {
			objectsCh <- minio.ObjectInfo{Key: k}
		}
	}
	close(objectsCh)

	var firstErr error
	for rmErr := range s.client.RemoveObjects(ctx, s.bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if rmErr.Err != nil && firstErr == nil {
			firstErr = rmErr.Err
		}
	}
	if firstErr != nil {
		return fmt.Errorf("remove minio objects: %w", firstErr)
	}
	return nil
}

func BuildMediaObjectKey(accountID, chatID, messageID int64, ext string) string {
	if ext == "" {
		ext = ".bin"
	}
	if ext[0] != '.' {
		ext = "." + ext
	}
	return fmt.Sprintf("accounts/%d/chats/%d/msg_%d%s", accountID, chatID, messageID, ext)
}

