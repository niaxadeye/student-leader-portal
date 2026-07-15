// Package config загружает конфигурацию из окружения (SITE.md §29).
package config

import "time"

type Config struct {
	App      App
	HTTP     HTTP
	Postgres Postgres
	Redis    Redis
	S3       S3
	JWT      JWT
	Cookie   Cookie
	Telegram Telegram
	Limits   Limits
	Features Features
	LogLevel string
}

type App struct {
	Env     string
	Name    string
	BaseURL string
	APIURL  string
}

type HTTP struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type Postgres struct {
	Host, Port, DB, User, Password, SSLMode string
}

type Redis struct {
	URL string
}

type S3 struct {
	Endpoint, Region, Bucket, AccessKey, SecretKey string
	UsePathStyle                                   bool
	PresignTTL                                     time.Duration
	// PublicEndpoint — хост, под которым presigned-URL отдаётся браузеру
	// (nginx проксирует его на внутренний MinIO). PublicSecure=true → https.
	PublicEndpoint string
	PublicSecure   bool
}

type JWT struct {
	Issuer, Audience, AccessSecret, RefreshSecret string
	AccessTTL, RefreshTTL                         time.Duration
}

type Cookie struct {
	Domain   string
	Secure   bool
	SameSite string
}

type Telegram struct {
	BotToken, DefaultChatID, DefaultThreadID string
	Enabled                                  bool
}

type Limits struct {
	MaxJSONBodyMB, MaxFileSizeMB, MaxSubmissionSizeMB int
}
