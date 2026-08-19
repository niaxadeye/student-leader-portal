package eventparticipants

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestVerifyTelegramLogin(t *testing.T) {
	t.Parallel()
	token := "123456:ABCDEF"
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	values := url.Values{}
	values.Set("id", "42")
	values.Set("first_name", "Ivan")
	values.Set("username", "durov")
	values.Set("auth_date", strconv.FormatInt(now.Unix(), 10))
	values.Set("hash", telegramLoginHash(token, values))

	got, err := verifyTelegramLogin(values, token, now)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.UserID != 42 || got.Username != "durov" {
		t.Fatalf("identity = %#v", got)
	}

	values.Set("hash", "deadbeef")
	if _, err := verifyTelegramLogin(values, token, now); err == nil {
		t.Fatal("bad hash must fail")
	}
}

func TestVerifyTelegramWebApp(t *testing.T) {
	t.Parallel()
	token := "123456:ABCDEF"
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(now.Unix(), 10))
	values.Set("user", `{"id":99,"username":"webapp"}`)
	values.Set("hash", telegramWebAppHash(token, values))

	got, err := verifyTelegramWebApp(values.Encode(), token, now)
	if err != nil {
		t.Fatalf("verify webapp: %v", err)
	}
	if got.UserID != 99 || got.Username != "webapp" {
		t.Fatalf("identity = %#v", got)
	}
}

func TestOAuthStateRoundTrip(t *testing.T) {
	t.Parallel()
	svc := &Service{social: SocialAuth{StateSecret: "state-secret"}, now: func() time.Time {
		return time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	}}
	now := svc.now()
	raw, err := svc.signOAuthState("event-2026", "telegram", now)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	slug, err := svc.parseOAuthState(raw, "telegram", now)
	if err != nil || slug != "event-2026" {
		t.Fatalf("parse = %q %v", slug, err)
	}
	if _, err := svc.parseOAuthState(raw, "vk", now); err == nil {
		t.Fatal("provider mismatch must fail")
	}
}

func telegramLoginHash(token string, values url.Values) string {
	secret := sha256.Sum256([]byte(token))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(telegramCheckString(values)))
	return hex.EncodeToString(mac.Sum(nil))
}

func telegramWebAppHash(token string, values url.Values) string {
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(token))
	secret := mac.Sum(nil)
	out := hmac.New(sha256.New, secret)
	out.Write([]byte(telegramCheckString(values)))
	return hex.EncodeToString(out.Sum(nil))
}

func telegramCheckString(values url.Values) string {
	_, check, _, _, err := parseTelegramPayload(values)
	if err != nil {
		panic(err)
	}
	return check
}

func TestLoginByTelegramMatchesUsernameAndBindsID(t *testing.T) {
	t.Parallel()
	token := "123456:ABCDEF"
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	p := activeParticipant()
	urlValue := "https://t.me/durov"
	p.TelegramURL = &urlValue
	repo := &fakeRepo{
		event:              activeEvent(),
		telegramByUsername: map[string]*Participant{"durov": &p},
	}
	svc := testService(repo, &fakeAudit{})
	svc.social = SocialAuth{TelegramBotToken: token, TelegramBotUsername: "testbot"}
	svc.now = func() time.Time { return now }

	values := url.Values{}
	values.Set("id", "42")
	values.Set("username", "durov")
	values.Set("auth_date", strconv.FormatInt(now.Unix(), 10))
	values.Set("hash", telegramLoginHash(token, values))

	result, err := svc.LoginByTelegramValues(context.Background(), "event-2026", values, ClientInfo{})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.Participant.ID != p.ID {
		t.Fatalf("participant = %q", result.Participant.ID)
	}
	if repo.boundTelegram == nil || *repo.boundTelegram != 42 {
		t.Fatalf("bind = %v", repo.boundTelegram)
	}
}
