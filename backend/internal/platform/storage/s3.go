// Package storage — тонкая обёртка над S3 для объектного хранилища файлов.
package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/eazytech/student-leader-cabinet/internal/config"
)

type Storage struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
	ttl       time.Duration
}

// New создаёт клиентов из конфига. Не обращается к сети до первого запроса.
func New(cfg config.S3) (*Storage, error) {
	if cfg.Endpoint == "" || cfg.Region == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("incomplete S3 configuration")
	}

	awsCfg := aws.Config{
		Region:      cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = cfg.UsePathStyle
	})

	return &Storage{
		client: client, presigner: s3.NewPresignClient(client),
		bucket: cfg.Bucket, ttl: cfg.PresignTTL,
	}, nil
}

// Put загружает объект и возвращает его ключ.
func (s *Storage) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), Body: r,
		ContentLength: aws.Int64(size), ContentType: aws.String(contentType),
	})
	return err
}

// Remove удаляет объект (идемпотентно — отсутствующий объект не ошибка).
func (s *Storage) Remove(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	return err
}

// PresignGet возвращает временную ссылку на скачивание, подписанную под публичный хост.
func (s *Storage) PresignGet(ctx context.Context, key string) (string, error) {
	request, err := s.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	}, s3.WithPresignExpires(s.ttl))
	if err != nil {
		return "", err
	}
	return request.URL, nil
}
