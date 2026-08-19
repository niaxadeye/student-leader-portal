// Package config загружает конфигурацию из окружения (SITE.md §29).
package config

import "time"

type Config struct {
	App             App
	HTTP            HTTP
	Postgres        Postgres
	S3              S3
	JWT             JWT
	Cookie          Cookie
	ParticipantAuth ParticipantAuth
	Telegram        Telegram
	VK              VK
	Limits          Limits
	Features        Features
	LogLevel        string
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

type S3 struct {
	Endpoint, Region, Bucket, AccessKey, SecretKey string
	UsePathStyle                                   bool
	PresignTTL                                     time.Duration
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

type ParticipantAuth struct {
	CookieName        string
	SessionTTL        time.Duration
	RateLimitWindow   time.Duration
	RateLimitAttempts int
	QRSecret          string
	QRTTL             time.Duration
}

type Telegram struct {
	BotToken, BotUsername, MiniAppName, DefaultChatID, DefaultThreadID string
	Enabled                                                            bool
}

type VK struct {
	ClientID, ClientSecret, RedirectURL, ServiceToken string
}

type Limits struct {
	MaxJSONBodyMB, MaxFileSizeMB, MaxSubmissionSizeMB int
}
