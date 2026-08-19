package eventparticipants

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

func jsonHTTP(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestParseFlexibleInt64(t *testing.T) {
	t.Parallel()
	got, err := parseFlexibleInt64(json.RawMessage(`"99"`))
	if err != nil || got != 99 {
		t.Fatalf("string id = %d %v", got, err)
	}
	got, err = parseFlexibleInt64(json.RawMessage(`101`))
	if err != nil || got != 101 {
		t.Fatalf("number id = %d %v", got, err)
	}
}

func TestLoginByVKAccessToken(t *testing.T) {
	t.Parallel()
	p := activeParticipant()
	repo := &fakeRepo{event: activeEvent(), vkByID: map[int64]*Participant{99: &p}}
	svc := testService(repo, &fakeAudit{})
	svc.social = SocialAuth{
		VKClientID: "54726553", VKRedirectURL: "https://eazytech.ru/login", VKServiceToken: "service",
	}
	svc.http = doerFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "user_info") {
			return jsonHTTP(`{"user":{"user_id":"99","first_name":"Ivan"}}`), nil
		}
		return jsonHTTP(`{"response":[{"id":99,"screen_name":"ivan"}]}`), nil
	})
	result, err := svc.LoginByVKAccessToken(context.Background(), "event-2026", "vk-access", ClientInfo{})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.Session == nil || result.Session.Participant.ID != p.ID {
		t.Fatalf("participant = %#v", result.Session)
	}
}

func TestFillVKUserIDResolvesScreenName(t *testing.T) {
	t.Parallel()
	svc := testService(&fakeRepo{}, &fakeAudit{})
	svc.social = SocialAuth{VKServiceToken: "service"}
	var asked string
	svc.http = doerFunc(func(req *http.Request) (*http.Response, error) {
		asked = req.URL.Query().Get("user_ids")
		return jsonHTTP(`{"response":[{"id":86867159,"screen_name":"erick_2dx2"}]}`), nil
	})

	screenNameURL := "https://vk.com/erick_2dx2"
	p := &Participant{VKURL: &screenNameURL}
	svc.fillVKUserID(context.Background(), p, nil, nil)
	if asked != "erick_2dx2" || p.VKUserID == nil || *p.VKUserID != 86867159 {
		t.Fatalf("asked=%q id=%v", asked, p.VKUserID)
	}

	// Числовую ссылку разбираем без обращения к API.
	asked = ""
	numericURL := "https://vk.com/id777"
	numeric := &Participant{VKURL: &numericURL}
	svc.fillVKUserID(context.Background(), numeric, nil, nil)
	if asked != "" || numeric.VKUserID == nil || *numeric.VKUserID != 777 {
		t.Fatalf("asked=%q id=%v", asked, numeric.VKUserID)
	}

	// Снятая ссылка снимает и привязку.
	previous := &Participant{VKURL: &screenNameURL, VKUserID: p.VKUserID}
	cleared := &Participant{}
	svc.fillVKUserID(context.Background(), cleared, previous, nil)
	if cleared.VKUserID != nil {
		t.Fatalf("cleared id = %v", cleared.VKUserID)
	}
}

func TestFillVKUserIDKeepsKnownIDWhenLookupFails(t *testing.T) {
	t.Parallel()
	svc := testService(&fakeRepo{}, &fakeAudit{})
	svc.social = SocialAuth{VKServiceToken: "service"}
	svc.http = doerFunc(func(*http.Request) (*http.Response, error) {
		return jsonHTTP(`{"error":{"error_code":5,"error_msg":"User authorization failed"}}`), nil
	})

	profileURL := "https://vk.com/erick_2dx2"
	known := int64(86867159)
	previous := &Participant{VKURL: &profileURL, VKUserID: &known}
	p := &Participant{VKURL: &profileURL}
	svc.fillVKUserID(context.Background(), p, previous, nil)
	if p.VKUserID == nil || *p.VKUserID != known {
		t.Fatalf("id = %v", p.VKUserID)
	}
}
