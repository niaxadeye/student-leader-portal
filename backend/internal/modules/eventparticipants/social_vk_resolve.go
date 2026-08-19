package eventparticipants

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type vkProfile struct {
	ID         int64  `json:"id"`
	ScreenName string `json:"screen_name"`
}

func (s *Service) vkUsersGetMany(ctx context.Context, slugs []string) vkIDCache {
	found := vkIDCache{}
	profiles, err := s.vkUsersGetAll(ctx, strings.Join(slugs, ","))
	if err != nil {
		return found
	}
	// VK возвращает профили в порядке запроса, но сопоставляем по screen_name.
	for index, profile := range profiles {
		if profile.ID <= 0 {
			continue
		}
		if screen := strings.ToLower(strings.TrimSpace(profile.ScreenName)); screen != "" {
			found[screen] = profile.ID
			continue
		}
		if index < len(slugs) {
			found[slugs[index]] = profile.ID
		}
	}
	return found
}

func (s *Service) vkUsersGet(ctx context.Context, userIDs string) (int64, string, error) {
	profiles, err := s.vkUsersGetAll(ctx, userIDs)
	if err != nil {
		return 0, "", err
	}
	return profiles[0].ID, profiles[0].ScreenName, nil
}

// Токен пользователя из VK ID привязан к IP браузера, поэтому обращаться
// к users.get с сервера можно только сервисным ключом приложения.
func (s *Service) vkUsersGetAll(ctx context.Context, userIDs string) ([]vkProfile, error) {
	token := strings.TrimSpace(s.social.VKServiceToken)
	if token == "" {
		slog.WarnContext(ctx, "vk_service_token_missing")
		return nil, ErrSocialUnavailable
	}
	if s.http == nil || strings.TrimSpace(userIDs) == "" {
		return nil, ErrInvalidCredentials
	}
	endpoint := "https://api.vk.com/method/users.get?" + url.Values{
		"user_ids":     {userIDs},
		"fields":       {"screen_name"},
		"access_token": {token},
		"v":            {"5.199"},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var payload struct {
		Response []vkProfile `json:"response"`
		Error    struct {
			Code int    `json:"error_code"`
			Msg  string `json:"error_msg"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || len(payload.Response) == 0 {
		slog.WarnContext(ctx, "vk_users_get_failed",
			"requested", len(strings.Split(userIDs, ",")), "status", resp.StatusCode,
			"vk_error_code", payload.Error.Code, "vk_error", payload.Error.Msg)
		return nil, ErrInvalidCredentials
	}
	return payload.Response, nil
}

// vkIDCache — заранее разрешённые слаги, чтобы импорт не ходил в VK построчно.
type vkIDCache map[string]int64

// resolveVKUserIDFromURL превращает ссылку на профиль в числовой id.
func (s *Service) resolveVKUserIDFromURL(ctx context.Context, profileURL *string, cache vkIDCache) *int64 {
	slug := vkSlugFromURL(profileURL)
	if slug == "" {
		return nil
	}
	if id, ok := vkNumericSlug(slug); ok {
		return &id
	}
	if id, ok := cache[slug]; ok && id > 0 {
		return &id
	}
	id, _, err := s.vkUsersGet(ctx, slug)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

// resolveVKSlugs разрешает до 300 коротких имён за один запрос к users.get.
func (s *Service) resolveVKSlugs(ctx context.Context, slugs []string) vkIDCache {
	cache := vkIDCache{}
	pending := make([]string, 0, len(slugs))
	seen := map[string]bool{}
	for _, slug := range slugs {
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		if _, ok := vkNumericSlug(slug); ok {
			continue
		}
		pending = append(pending, slug)
	}
	const batchSize = 300
	for start := 0; start < len(pending); start += batchSize {
		end := min(start+batchSize, len(pending))
		batch := pending[start:end]
		for slug, id := range s.vkUsersGetMany(ctx, batch) {
			cache[slug] = id
		}
	}
	return cache
}

// fillVKUserID хранит числовой id профиля: короткое имя владелец может сменить,
// а вход по VK сопоставляется именно по id.
func (s *Service) fillVKUserID(ctx context.Context, p, existing *Participant, cache vkIDCache) {
	if p.VKURL == nil {
		p.VKUserID = nil
		return
	}
	unchanged := existing != nil && sameOptionalString(existing.VKURL, p.VKURL)
	if unchanged && existing.VKUserID != nil {
		p.VKUserID = existing.VKUserID
		return
	}
	if resolved := s.resolveVKUserIDFromURL(ctx, p.VKURL, cache); resolved != nil {
		p.VKUserID = resolved
		return
	}
	if unchanged {
		p.VKUserID = existing.VKUserID
	}
}

func sameOptionalString(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return strings.EqualFold(*a, *b)
}

// vkSlugFromURL достаёт последний сегмент пути: id123 или короткое имя.
func vkSlugFromURL(profileURL *string) string {
	if profileURL == nil {
		return ""
	}
	raw := strings.TrimSpace(*profileURL)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	slug := strings.Trim(parsed.Path, "/")
	if slug == "" || strings.Contains(slug, "/") {
		return ""
	}
	return strings.ToLower(slug)
}

func vkNumericSlug(slug string) (int64, bool) {
	if !strings.HasPrefix(slug, "id") {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(slug, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
