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
	svc.social = SocialAuth{VKClientID: "54726553", VKRedirectURL: "https://eazytech.ru/login"}
	svc.http = doerFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "user_info") {
			return jsonHTTP(`{"user":{"user_id":"99","first_name":"Ivan"}}`), nil
		}
		return jsonHTTP(`{"response":[{"screen_name":"ivan"}]}`), nil
	})
	result, err := svc.LoginByVKAccessToken(context.Background(), "event-2026", "vk-access", ClientInfo{})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if result.Session == nil || result.Session.Participant.ID != p.ID {
		t.Fatalf("participant = %#v", result.Session)
	}
}
