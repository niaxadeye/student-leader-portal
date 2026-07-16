package config

import (
	"os"
	"strconv"
	"time"
)

// Load читает конфигурацию из окружения. Переменные из .env подхватываются
// через окружение процесса (systemd EnvironmentFile или `make` через env).
func Load() (*Config, error) {
	return &Config{
		App: App{
			Env:     env("APP_ENV", "development"),
			Name:    env("APP_NAME", "student-leader-cabinet"),
			BaseURL: env("APP_BASE_URL", "http://localhost:5173"),
			APIURL:  env("API_BASE_URL", "http://localhost:8080"),
		},
		HTTP: HTTP{
			Host:         env("HTTP_HOST", "127.0.0.1"),
			Port:         env("HTTP_PORT", "8080"),
			// Таймауты подняты под загрузку крупных файлов (до DEFAULT_MAX_FILE_SIZE_MB) на медленных каналах.
			ReadTimeout:  envDur("HTTP_READ_TIMEOUT", 20*time.Minute),
			WriteTimeout: envDur("HTTP_WRITE_TIMEOUT", 20*time.Minute),
			IdleTimeout:  envDur("HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		Postgres: Postgres{
			Host: env("POSTGRES_HOST", "127.0.0.1"), Port: env("POSTGRES_PORT", "5432"),
			DB: env("POSTGRES_DB", "student_leader"), User: env("POSTGRES_USER", "student_leader"),
			Password: env("POSTGRES_PASSWORD", ""), SSLMode: env("POSTGRES_SSLMODE", "disable"),
		},
		Redis: Redis{URL: env("REDIS_URL", "redis://127.0.0.1:6379/0")},
		S3: S3{
			Endpoint: env("S3_ENDPOINT", "http://127.0.0.1:9000"), Region: env("S3_REGION", "us-east-1"),
			Bucket: env("S3_BUCKET", "student-leader-files"), AccessKey: env("S3_ACCESS_KEY", "minio"),
			SecretKey: env("S3_SECRET_KEY", ""), UsePathStyle: envBool("S3_USE_PATH_STYLE", true),
			PresignTTL:     envDur("S3_PRESIGN_TTL", 24*time.Hour),
			PublicEndpoint: env("S3_PUBLIC_ENDPOINT", "eazytech.ru"),
			PublicSecure:   envBool("S3_PUBLIC_SECURE", true),
		},
		JWT: JWT{
			Issuer: env("JWT_ISSUER", "student-leader-cabinet"), Audience: env("JWT_AUDIENCE", "student-leader-web"),
			AccessSecret: env("JWT_ACCESS_SECRET", ""), RefreshSecret: env("JWT_REFRESH_SECRET", ""),
			AccessTTL: envDur("ACCESS_TOKEN_TTL", 15*time.Minute), RefreshTTL: envDur("REFRESH_TOKEN_TTL", 720*time.Hour),
		},
		Cookie: Cookie{
			Domain: env("COOKIE_DOMAIN", "localhost"), Secure: envBool("COOKIE_SECURE", false),
			SameSite: env("COOKIE_SAMESITE", "lax"),
		},
		Telegram: Telegram{
			BotToken: env("TELEGRAM_BOT_TOKEN", ""), DefaultChatID: env("TELEGRAM_DEFAULT_CHAT_ID", ""),
			DefaultThreadID: env("TELEGRAM_DEFAULT_THREAD_ID", ""), Enabled: envBool("TELEGRAM_NOTIFICATIONS_ENABLED", false),
		},
		Limits: Limits{
			MaxJSONBodyMB: envInt("MAX_JSON_BODY_MB", 2), MaxFileSizeMB: envInt("DEFAULT_MAX_FILE_SIZE_MB", 1024),
			MaxSubmissionSizeMB: envInt("DEFAULT_MAX_SUBMISSION_SIZE_MB", 10240),
		},
		Features: loadFeatures(),
		LogLevel: env("LOG_LEVEL", "info"),
	}, nil
}

func (p Postgres) DSN() string {
	return "postgres://" + p.User + ":" + p.Password + "@" + p.Host + ":" + p.Port + "/" + p.DB + "?sslmode=" + p.SSLMode
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envDur(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
