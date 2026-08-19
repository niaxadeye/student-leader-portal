package eventparticipants

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type oauthState struct {
	EventSlug string `json:"e"`
	Provider  string `json:"p"`
	Expires   int64  `json:"x"`
}

func (s *Service) signOAuthState(eventSlug, provider string, now time.Time) (string, error) {
	if strings.TrimSpace(s.social.StateSecret) == "" {
		return "", ErrSocialUnavailable
	}
	payload, err := json.Marshal(oauthState{
		EventSlug: eventSlug, Provider: provider, Expires: now.Add(10 * time.Minute).Unix(),
	})
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(s.social.StateSecret))
	mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *Service) parseOAuthState(raw, provider string, now time.Time) (string, error) {
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, ".")
	if len(parts) != 2 || strings.TrimSpace(s.social.StateSecret) == "" {
		return "", ErrInvalidCredentials
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInvalidCredentials
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrInvalidCredentials
	}
	mac := hmac.New(sha256.New, []byte(s.social.StateSecret))
	mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", ErrInvalidCredentials
	}
	var state oauthState
	if err := json.Unmarshal(payload, &state); err != nil {
		return "", ErrInvalidCredentials
	}
	if state.Provider != provider || strings.TrimSpace(state.EventSlug) == "" || now.Unix() > state.Expires {
		return "", ErrInvalidCredentials
	}
	return state.EventSlug, nil
}

func telegramBotID(token string) string {
	id, _, _ := strings.Cut(strings.TrimSpace(token), ":")
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
		return ""
	}
	return id
}
