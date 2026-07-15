package outbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Telegram отправляет сообщения через Bot API sendMessage.
type Telegram struct {
	token         string
	defaultChat   string
	defaultThread string
	enabled       bool
	baseURL       string
	http          *http.Client
}

// TelegramConfig — параметры из окружения (SITE.md §22).
type TelegramConfig struct {
	Token         string
	DefaultChat   string
	DefaultThread string
	Enabled       bool
	// BaseURL переопределяется в тестах; по умолчанию https://api.telegram.org.
	BaseURL string
}

// NewTelegramSender строит Telegram-отправителя. Считается включённым только при
// Enabled=true И заданных токене/чате — иначе диспетчер копит события, но не шлёт.
func NewTelegramSender(cfg TelegramConfig) *Telegram {
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.telegram.org"
	}
	return &Telegram{
		token:         cfg.Token,
		defaultChat:   cfg.DefaultChat,
		defaultThread: cfg.DefaultThread,
		enabled:       cfg.Enabled && cfg.Token != "" && cfg.DefaultChat != "",
		baseURL:       base,
		http:          &http.Client{Timeout: 15 * time.Second},
	}
}

// Enabled сообщает диспетчеру, можно ли слать (иначе события копятся как PENDING).
func (t *Telegram) Enabled() bool { return t.enabled }

// Send шлёт текст в чат по умолчанию. Возвращает ошибку при не-2xx или сетевом сбое
// (вызывающий решает про retry/backoff).
func (t *Telegram) Send(ctx context.Context, text string) error {
	body := map[string]any{
		"chat_id":    t.defaultChat,
		"text":       text,
		"parse_mode": "HTML",
		// Ссылки в тексте не нужно превращать в превью.
		"disable_web_page_preview": true,
	}
	if t.defaultThread != "" {
		body["message_thread_id"] = t.defaultThread
	}
	raw, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/bot%s/sendMessage", t.baseURL, t.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return fmt.Errorf("telegram %d: %s", resp.StatusCode, msg)
	}
	return nil
}
