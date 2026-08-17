package storage

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/eazytech/student-leader-cabinet/internal/config"
)

func TestNewRejectsIncompleteConfig(t *testing.T) {
	_, err := New(config.S3{Endpoint: "https://s3.twcstorage.ru", Region: "ru-1"})
	if err == nil {
		t.Fatal("New() error = nil, want incomplete configuration error")
	}
}

func TestPresignGetUsesConfiguredPathStyleEndpoint(t *testing.T) {
	store, err := New(config.S3{
		Endpoint: "https://s3.twcstorage.ru", Region: "ru-1",
		Bucket: "test-bucket", AccessKey: "access", SecretKey: "secret",
		UsePathStyle: true, PresignTTL: 15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	rawURL, err := store.PresignGet(context.Background(), "tasks/example/image.png")
	if err != nil {
		t.Fatalf("PresignGet() error = %v", err)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if parsed.Scheme != "https" || parsed.Host != "s3.twcstorage.ru" {
		t.Fatalf("presigned endpoint = %s://%s, want https://s3.twcstorage.ru", parsed.Scheme, parsed.Host)
	}
	if parsed.Path != "/test-bucket/tasks/example/image.png" {
		t.Fatalf("presigned path = %q, want path-style bucket and key", parsed.Path)
	}
	if got := parsed.Query().Get("X-Amz-Expires"); got != "900" {
		t.Fatalf("X-Amz-Expires = %q, want 900", got)
	}
}
