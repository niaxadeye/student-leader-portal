package eventparticipants

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type LoginOptions struct {
	Telegram struct {
		Enabled     bool   `json:"enabled"`
		BotUsername string `json:"bot_username,omitempty"`
	} `json:"telegram"`
	VK struct {
		Enabled     bool   `json:"enabled"`
		AppID       string `json:"app_id,omitempty"`
		RedirectURL string `json:"redirect_url,omitempty"`
	} `json:"vk"`
	Events []PublicEvent `json:"events"`
}

type PublicEvent struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type vkIdentity struct {
	UserID     int64
	ScreenName string
}

func (s *Service) LoginOptions(ctx context.Context) (*LoginOptions, error) {
	options := &LoginOptions{}
	options.Telegram.Enabled = s.social.TelegramEnabled()
	options.Telegram.BotUsername = strings.TrimPrefix(strings.TrimSpace(s.social.TelegramBotUsername), "@")
	options.VK.Enabled = s.social.VKEnabled()
	if options.VK.Enabled {
		options.VK.AppID = strings.TrimSpace(s.social.VKClientID)
		options.VK.RedirectURL = strings.TrimSpace(s.social.VKRedirectURL)
	}
	events, err := s.repo.ListActiveEvents(ctx)
	if err != nil {
		return nil, err
	}
	options.Events = make([]PublicEvent, 0, len(events))
	for _, event := range events {
		options.Events = append(options.Events, PublicEvent{Slug: event.Slug, Name: event.Name})
	}
	return options, nil
}

func (s *Service) VKStartURL(eventSlug string, now time.Time) (string, string, error) {
	if !s.social.VKEnabled() {
		return "", "", ErrSocialUnavailable
	}
	state, err := s.signOAuthState(strings.TrimSpace(eventSlug), "vk", now)
	if err != nil {
		return "", "", err
	}
	values := url.Values{}
	values.Set("client_id", s.social.VKClientID)
	values.Set("display", "page")
	values.Set("redirect_uri", s.social.VKRedirectURL)
	values.Set("response_type", "code")
	values.Set("scope", "")
	values.Set("state", state)
	values.Set("v", "5.199")
	return "https://oauth.vk.com/authorize?" + values.Encode(), state, nil
}

// LoginByTelegramValues — вход с сайта через Telegram Login Widget.
func (s *Service) LoginByTelegramValues(
	ctx context.Context,
	eventSlug string,
	values url.Values,
	client ClientInfo,
) (*SocialAuthResult, error) {
	if !s.social.TelegramEnabled() {
		return nil, ErrSocialUnavailable
	}
	identity, err := verifyTelegramLogin(values, s.social.TelegramBotToken, s.now().UTC())
	if err != nil {
		return nil, err
	}
	return s.loginByTelegramIdentity(ctx, eventSlug, false, identity, client, "telegram")
}

func (s *Service) LoginByTelegramWebApp(
	ctx context.Context,
	eventSlug, initData string,
	client ClientInfo,
) (*SocialAuthResult, error) {
	if !s.social.TelegramEnabled() {
		return nil, ErrSocialUnavailable
	}
	identity, err := verifyTelegramWebApp(strings.TrimSpace(initData), s.social.TelegramBotToken, s.now().UTC())
	if err != nil {
		return nil, err
	}
	return s.loginByTelegramIdentity(ctx, eventSlug, false, identity, client, "telegram_webapp")
}

func (s *Service) LoginByVKAccessToken(
	ctx context.Context,
	eventSlug, accessToken string,
	client ClientInfo,
) (*SocialAuthResult, error) {
	if !s.social.VKEnabled() {
		return nil, ErrSocialUnavailable
	}
	identity, err := s.userInfoFromVKAccessToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return s.loginByVKIdentity(ctx, eventSlug, false, identity, client)
}

func (s *Service) LoginByVKCallback(
	ctx context.Context,
	code, state string,
	client ClientInfo,
) (*SocialAuthResult, error) {
	if !s.social.VKEnabled() {
		return nil, ErrSocialUnavailable
	}
	eventSlug, err := s.parseOAuthState(state, "vk", s.now())
	if err != nil {
		return nil, err
	}
	identity, err := s.exchangeVKCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.loginByVKIdentity(ctx, eventSlug, false, identity, client)
}

func (s *Service) ContinueSocialLogin(
	ctx context.Context,
	token, eventSlug string,
	client ClientInfo,
) (*SocialAuthResult, error) {
	cont, err := s.parseSocialContinue(token, s.now())
	if err != nil {
		return nil, err
	}
	eventSlug = strings.TrimSpace(eventSlug)
	switch cont.Provider {
	case "telegram", "telegram_webapp":
		return s.loginByTelegramIdentity(ctx, eventSlug, eventSlug != "", telegramIdentity{
			UserID: cont.UserID, Username: cont.Extra,
		}, client, cont.Provider)
	case "vk":
		return s.loginByVKIdentity(ctx, eventSlug, eventSlug != "", vkIdentity{
			UserID: cont.UserID, ScreenName: cont.Extra,
		}, client)
	default:
		return nil, ErrInvalidCredentials
	}
}

func (s *Service) loginByTelegramIdentity(
	ctx context.Context,
	eventSlug string,
	requirePreferred bool,
	identity telegramIdentity,
	client ClientInfo,
	method string,
) (*SocialAuthResult, error) {
	if identity.UserID <= 0 {
		return nil, ErrInvalidCredentials
	}
	matches, err := s.repo.ListActiveByTelegramUserID(ctx, identity.UserID)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 && identity.Username != "" {
		matches, err = s.repo.ListActiveByTelegramUsername(ctx, identity.Username)
		if err != nil {
			return nil, err
		}
	}
	return s.resolveSocialMatches(ctx, matches, eventSlug, requirePreferred, client, method, func(match ParticipantEventMatch) error {
		return s.repo.BindTelegram(ctx, match.Event.ID, match.Participant.ID, identity.UserID, canonicalTelegramURL(identity.Username))
	}, method, identity.UserID, identity.Username)
}

func (s *Service) loginByVKIdentity(
	ctx context.Context,
	eventSlug string,
	requirePreferred bool,
	identity vkIdentity,
	client ClientInfo,
) (*SocialAuthResult, error) {
	if identity.UserID <= 0 {
		return nil, ErrInvalidCredentials
	}
	matches, err := s.repo.ListActiveByVKUserID(ctx, identity.UserID)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		matches, err = s.repo.ListActiveByVKIdentity(ctx, identity.UserID, identity.ScreenName)
		if err != nil {
			return nil, err
		}
	}
	return s.resolveSocialMatches(ctx, matches, eventSlug, requirePreferred, client, "vk", func(match ParticipantEventMatch) error {
		return s.repo.BindVK(ctx, match.Event.ID, match.Participant.ID, identity.UserID, canonicalVKURL(identity.UserID, identity.ScreenName))
	}, "vk", identity.UserID, identity.ScreenName)
}

func (s *Service) resolveSocialMatches(
	ctx context.Context,
	matches []ParticipantEventMatch,
	preferredSlug string,
	requirePreferred bool,
	client ClientInfo,
	method string,
	bind func(ParticipantEventMatch) error,
	provider string,
	userID int64,
	extra string,
) (*SocialAuthResult, error) {
	if len(matches) == 0 {
		// Без этого «участник не найден» невозможно отличить от несовпадения ссылок.
		slog.WarnContext(ctx, "social_login_no_match",
			"provider", provider, "social_user_id", userID, "social_identity", extra)
		return nil, ErrSocialNotLinked
	}
	preferredSlug = strings.TrimSpace(preferredSlug)
	var chosen *ParticipantEventMatch
	if preferredSlug != "" {
		for i := range matches {
			if matches[i].Event.Slug == preferredSlug {
				chosen = &matches[i]
				break
			}
		}
		if chosen == nil && requirePreferred {
			return nil, ErrInvalidCredentials
		}
	}
	if chosen == nil && len(matches) == 1 {
		chosen = &matches[0]
	}
	if chosen != nil {
		_ = bind(*chosen)
		session, err := s.issueSession(ctx, &chosen.Event, &chosen.Participant, client, method)
		if err != nil {
			return nil, err
		}
		return &SocialAuthResult{Session: session}, nil
	}
	token, err := s.signSocialContinue(provider, userID, extra, s.now())
	if err != nil {
		return nil, err
	}
	events := make([]PublicEvent, 0, len(matches))
	for _, match := range matches {
		events = append(events, PublicEvent{Slug: match.Event.Slug, Name: match.Event.Name})
	}
	return &SocialAuthResult{Events: events, ContinueToken: token}, nil
}

func canonicalTelegramURL(username string) *string {
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if username == "" {
		return nil
	}
	value := "https://t.me/" + username
	return &value
}

func canonicalVKURL(userID int64, screenName string) *string {
	screenName = strings.TrimSpace(screenName)
	value := "https://vk.com/id" + strconv.FormatInt(userID, 10)
	if screenName != "" && !strings.HasPrefix(strings.ToLower(screenName), "id") {
		value = "https://vk.com/" + screenName
	}
	return &value
}

func (s *Service) userInfoFromVKAccessToken(ctx context.Context, accessToken string) (vkIdentity, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" || s.http == nil {
		return vkIdentity{}, ErrInvalidCredentials
	}
	form := url.Values{}
	form.Set("access_token", accessToken)
	if id := strings.TrimSpace(s.social.VKClientID); id != "" {
		form.Set("client_id", id)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://id.vk.ru/oauth2/user_info", strings.NewReader(form.Encode()))
	if err != nil {
		return vkIdentity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.http.Do(req)
	if err != nil {
		return vkIdentity{}, fmt.Errorf("vk user_info: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var payload struct {
		User struct {
			UserID    json.RawMessage `json:"user_id"`
			FirstName string          `json:"first_name"`
			LastName  string          `json:"last_name"`
		} `json:"user"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Error != "" {
		return vkIdentity{}, ErrInvalidCredentials
	}
	userID, err := parseFlexibleInt64(payload.User.UserID)
	if err != nil || userID <= 0 {
		return vkIdentity{}, ErrInvalidCredentials
	}
	identity := vkIdentity{UserID: userID}
	identity.ScreenName = s.vkScreenName(ctx, userID)
	return identity, nil
}

func parseFlexibleInt64(raw json.RawMessage) (int64, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return 0, ErrInvalidCredentials
	}
	if raw[0] == '"' {
		var asString string
		if err := json.Unmarshal(raw, &asString); err != nil {
			return 0, err
		}
		return strconv.ParseInt(strings.TrimSpace(asString), 10, 64)
	}
	var asNumber int64
	if err := json.Unmarshal(raw, &asNumber); err != nil {
		return 0, err
	}
	return asNumber, nil
}

func (s *Service) exchangeVKCode(ctx context.Context, code string) (vkIdentity, error) {
	code = strings.TrimSpace(code)
	if code == "" || s.http == nil {
		return vkIdentity{}, ErrInvalidCredentials
	}
	tokenURL := "https://oauth.vk.com/access_token?" + url.Values{
		"client_id":     {s.social.VKClientID},
		"client_secret": {s.social.VKClientSecret},
		"redirect_uri":  {s.social.VKRedirectURL},
		"code":          {code},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return vkIdentity{}, err
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return vkIdentity{}, fmt.Errorf("vk token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var token struct {
		UserID           int64  `json:"user_id"`
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &token); err != nil || token.UserID <= 0 || token.AccessToken == "" {
		slog.WarnContext(ctx, "vk_code_exchange_failed",
			"status", resp.StatusCode, "vk_error", token.Error, "vk_error_description", token.ErrorDescription)
		return vkIdentity{}, ErrInvalidCredentials
	}
	identity := vkIdentity{UserID: token.UserID}
	identity.ScreenName = s.vkScreenName(ctx, token.UserID)
	return identity, nil
}

// vkScreenName нужен, чтобы сопоставить проверенный id со ссылкой из базы.
func (s *Service) vkScreenName(ctx context.Context, userID int64) string {
	_, screenName, err := s.vkUsersGet(ctx, strconv.FormatInt(userID, 10))
	if err != nil {
		return ""
	}
	return screenName
}
