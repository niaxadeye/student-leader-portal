package eventparticipants

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const socialAuthMaxAge = 24 * time.Hour

type telegramIdentity struct {
	UserID   int64
	Username string
}

// telegramValuesFromAuthResult разбирает base64 из фрагмента #tgAuthResult.
// Все поля переносятся как есть: подпись Telegram считается по всему набору.
func telegramValuesFromAuthResult(raw string) (url.Values, error) {
	data, err := decodeLooseBase64(strings.TrimSpace(raw))
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var payload map[string]any
	if err := decoder.Decode(&payload); err != nil {
		return nil, ErrInvalidCredentials
	}
	values := url.Values{}
	for key, value := range payload {
		switch typed := value.(type) {
		case nil:
			continue
		case string:
			values.Set(key, typed)
		case json.Number:
			values.Set(key, typed.String())
		case bool:
			values.Set(key, strconv.FormatBool(typed))
		default:
			encoded, err := json.Marshal(typed)
			if err != nil {
				return nil, ErrInvalidCredentials
			}
			values.Set(key, string(encoded))
		}
	}
	return values, nil
}

func decodeLooseBase64(raw string) ([]byte, error) {
	unpadded := strings.TrimRight(raw, "=")
	for _, encoding := range []*base64.Encoding{base64.RawURLEncoding, base64.RawStdEncoding} {
		if data, err := encoding.DecodeString(unpadded); err == nil {
			return data, nil
		}
	}
	return nil, ErrInvalidCredentials
}

func verifyTelegramLogin(values url.Values, botToken string, now time.Time) (telegramIdentity, error) {
	identity, checkString, hash, authDate, err := parseTelegramPayload(values)
	if err != nil || hash == "" {
		return telegramIdentity{}, ErrInvalidCredentials
	}
	secret := sha256.Sum256([]byte(botToken))
	if !hmacSHA256Hex(secret[:], checkString, hash) {
		return telegramIdentity{}, ErrInvalidCredentials
	}
	if now.Sub(authDate) > socialAuthMaxAge || authDate.After(now.Add(5*time.Minute)) {
		return telegramIdentity{}, ErrInvalidCredentials
	}
	return identity, nil
}

func verifyTelegramWebApp(initData, botToken string, now time.Time) (telegramIdentity, error) {
	values, err := url.ParseQuery(initData)
	if err != nil {
		return telegramIdentity{}, ErrInvalidCredentials
	}
	identity, checkString, hash, authDate, err := parseTelegramPayload(values)
	if err != nil || hash == "" {
		return telegramIdentity{}, ErrInvalidCredentials
	}
	mac := hmac.New(sha256.New, []byte("WebAppData"))
	mac.Write([]byte(botToken))
	if !hmacSHA256Hex(mac.Sum(nil), checkString, hash) {
		return telegramIdentity{}, ErrInvalidCredentials
	}
	if now.Sub(authDate) > socialAuthMaxAge || authDate.After(now.Add(5*time.Minute)) {
		return telegramIdentity{}, ErrInvalidCredentials
	}
	if identity.UserID == 0 {
		userJSON := values.Get("user")
		identity, err = parseTelegramWebAppUser(userJSON)
		if err != nil {
			return telegramIdentity{}, err
		}
	}
	return identity, nil
}

func parseTelegramPayload(values url.Values) (telegramIdentity, string, string, time.Time, error) {
	hash := strings.TrimSpace(values.Get("hash"))
	keys := make([]string, 0, len(values))
	for key := range values {
		if key == "hash" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	authUnix, err := strconv.ParseInt(values.Get("auth_date"), 10, 64)
	if err != nil || authUnix <= 0 {
		return telegramIdentity{}, "", "", time.Time{}, ErrInvalidCredentials
	}
	identity := telegramIdentity{Username: strings.TrimSpace(values.Get("username"))}
	if id := strings.TrimSpace(values.Get("id")); id != "" {
		identity.UserID, err = strconv.ParseInt(id, 10, 64)
		if err != nil || identity.UserID <= 0 {
			return telegramIdentity{}, "", "", time.Time{}, ErrInvalidCredentials
		}
	}
	return identity, strings.Join(parts, "\n"), hash, time.Unix(authUnix, 0).UTC(), nil
}

func parseTelegramWebAppUser(raw string) (telegramIdentity, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return telegramIdentity{}, ErrInvalidCredentials
	}
	// Mini App передаёт user как JSON-строку в initData.
	id := jsonInt64Field(raw, "id")
	if id <= 0 {
		return telegramIdentity{}, ErrInvalidCredentials
	}
	return telegramIdentity{UserID: id, Username: jsonStringField(raw, "username")}, nil
}

func jsonInt64Field(raw, key string) int64 {
	needle := `"` + key + `":`
	index := strings.Index(raw, needle)
	if index < 0 {
		return 0
	}
	value := strings.TrimLeft(raw[index+len(needle):], " ")
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	n, err := strconv.ParseInt(value[:end], 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func jsonStringField(raw, key string) string {
	needle := `"` + key + `":"`
	index := strings.Index(raw, needle)
	if index < 0 {
		return ""
	}
	value := raw[index+len(needle):]
	end := strings.IndexByte(value, '"')
	if end < 0 {
		return ""
	}
	return value[:end]
}

func hmacSHA256Hex(key []byte, message, wantHex string) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	got, err := hex.DecodeString(strings.TrimSpace(wantHex))
	if err != nil || len(got) != sha256.Size {
		return false
	}
	return hmac.Equal(got, mac.Sum(nil))
}
