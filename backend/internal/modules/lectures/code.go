package lectures

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"time"
)

type codePayload struct {
	Version int    `json:"v"`
	Nonce   string `json:"n"`
	Expires int64  `json:"exp"`
}

// CodeManager signs short-lived opaque QR tokens. Participant/event identifiers are
// deliberately absent from the client-visible payload; the nonce is resolved in DB.
type CodeManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
	random io.Reader
}

func NewCodeManager(secret string, ttl time.Duration) *CodeManager {
	if ttl <= 0 {
		ttl = 45 * time.Second
	}
	return &CodeManager{secret: []byte(secret), ttl: ttl, now: time.Now, random: rand.Reader}
}

func (m *CodeManager) New() (token, nonceHash string, expiresAt time.Time, err error) {
	raw := make([]byte, 32)
	if _, err = io.ReadFull(m.random, raw); err != nil {
		return "", "", time.Time{}, err
	}
	nonce := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt = m.now().UTC().Add(m.ttl)
	payload, err := json.Marshal(codePayload{Version: 1, Nonce: nonce, Expires: expiresAt.Unix()})
	if err != nil {
		return "", "", time.Time{}, err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	token = encoded + "." + m.sign(encoded)
	nonceHash = hashNonce(nonce)
	return token, nonceHash, expiresAt, nil
}

func (m *CodeManager) Verify(token string) (VerifiedCode, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || len(m.secret) == 0 {
		return VerifiedCode{}, ErrInvalidCode
	}
	expected := m.sign(parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return VerifiedCode{}, ErrInvalidCode
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return VerifiedCode{}, ErrInvalidCode
	}
	var payload codePayload
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Version != 1 || payload.Nonce == "" {
		return VerifiedCode{}, ErrInvalidCode
	}
	expiresAt := time.Unix(payload.Expires, 0).UTC()
	if !expiresAt.After(m.now()) {
		return VerifiedCode{}, ErrExpiredCode
	}
	return VerifiedCode{NonceHash: hashNonce(payload.Nonce), ExpiresAt: expiresAt}, nil
}

func (m *CodeManager) TTL() time.Duration { return m.ttl }

func (m *CodeManager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func hashNonce(nonce string) string {
	sum := sha256.Sum256([]byte(nonce))
	return hex.EncodeToString(sum[:])
}
