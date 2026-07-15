// Package storage — тонкая обёртка над MinIO/S3 для объектного хранилища файлов.
// Держит два клиента: internal (запись/удаление от API) и public (presigned-URL,
// подписанные под публичный хост, который nginx проксирует на внутренний MinIO).
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/eazytech/student-leader-cabinet/internal/config"
)

type Storage struct {
	internal *minio.Client
	public   *minio.Client
	bucket   string
	ttl      time.Duration
}

// hostFromEndpoint убирает схему из endpoint (minio-go хочет host[:port]).
func hostFromEndpoint(endpoint string) (host string, secure bool) {
	secure = strings.HasPrefix(endpoint, "https://")
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		return u.Host, secure
	}
	return strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://"), secure
}

// New создаёт клиентов из конфига. Не обращается к сети до первого запроса.
func New(cfg config.S3) (*Storage, error) {
	host, secure := hostFromEndpoint(cfg.Endpoint)
	creds := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")
	internal, err := minio.New(host, &minio.Options{Creds: creds, Secure: secure, Region: cfg.Region})
	if err != nil {
		return nil, fmt.Errorf("minio internal: %w", err)
	}
	public, err := minio.New(cfg.PublicEndpoint, &minio.Options{
		Creds: creds, Secure: cfg.PublicSecure, Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio public: %w", err)
	}
	return &Storage{internal: internal, public: public, bucket: cfg.Bucket, ttl: cfg.PresignTTL}, nil
}

// Put загружает объект и возвращает его ключ.
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.internal.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

// Remove удаляет объект (идемпотентно — отсутствующий объект не ошибка).
func (s *Storage) Remove(ctx context.Context, key string) error {
	return s.internal.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

// PresignGet возвращает временную ссылку на скачивание, подписанную под публичный хост.
func (s *Storage) PresignGet(ctx context.Context, key string) (string, error) {
	u, err := s.public.PresignedGetObject(ctx, s.bucket, key, s.ttl, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
