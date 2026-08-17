package eventparticipants

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

func (s *Service) LoginByName(
	ctx context.Context,
	eventSlug, fullName string,
	birthDate time.Time,
	client ClientInfo,
) (*SessionResult, error) {
	event, err := s.loginEvent(ctx, eventSlug)
	if err != nil {
		return nil, err
	}
	normalized := NormalizeFullName(fullName)
	if normalized == "" || birthDate.IsZero() {
		return nil, ErrInvalidCredentials
	}
	matches, err := s.repo.FindByNameBirth(ctx, event.ID, normalized, birthDate)
	if err != nil {
		return nil, err
	}
	if len(matches) > 1 {
		return nil, ErrAmbiguousIdentity
	}
	if len(matches) == 0 {
		return nil, ErrInvalidCredentials
	}
	return s.issueSession(ctx, event, &matches[0], client, "fio_birth_date")
}

func (s *Service) LoginByUnionCard(ctx context.Context, eventSlug, number string, client ClientInfo) (*SessionResult, error) {
	return s.loginByIdentifier(ctx, eventSlug, number, client, "union_card", s.repo.FindByUnionCard)
}

func (s *Service) LoginBySKSBarcode(ctx context.Context, eventSlug, barcode string, client ClientInfo) (*SessionResult, error) {
	return s.loginByIdentifier(ctx, eventSlug, barcode, client, "sks_barcode", s.repo.FindBySKSBarcode)
}

type identifierLookup func(context.Context, string, string) (*Participant, error)

func (s *Service) loginByIdentifier(
	ctx context.Context,
	eventSlug, identifier string,
	client ClientInfo,
	method string,
	lookup identifierLookup,
) (*SessionResult, error) {
	event, err := s.loginEvent(ctx, eventSlug)
	if err != nil {
		return nil, err
	}
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return nil, ErrInvalidCredentials
	}
	p, err := lookup(ctx, event.ID, identifier)
	if err != nil {
		if err == ErrNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	return s.issueSession(ctx, event, p, client, method)
}

func (s *Service) loginEvent(ctx context.Context, slug string) (*EventRef, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, ErrInvalidCredentials
	}
	event, err := s.repo.EventBySlug(ctx, slug)
	if errors.Is(err, ErrEventUnavailable) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if event.Status != "ACTIVE" {
		return nil, ErrInvalidCredentials
	}
	return event, nil
}

func (s *Service) issueSession(
	ctx context.Context,
	event *EventRef,
	participant *Participant,
	client ClientInfo,
	method string,
) (*SessionResult, error) {
	if participant.Status != StatusActive {
		return nil, ErrInvalidCredentials
	}
	token, err := s.newToken()
	if err != nil {
		return nil, err
	}
	expiresAt := s.now().Add(s.sessionTTL)
	_, err = s.repo.CreateSession(ctx, event.ID, participant.ID, security.HashToken(token),
		client.UserAgent, security.HashIP(client.IP), expiresAt)
	if err != nil {
		return nil, err
	}
	s.audit.LogParticipant(ctx, participant.ID, event.ID, "PARTICIPANT_LOGIN", "participant_session", "",
		map[string]any{"method": method})
	return &SessionResult{Token: token, ExpiresAt: expiresAt, Participant: *participant, Event: *event}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (*Principal, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrSessionExpired
	}
	return s.repo.AuthenticateSession(ctx, security.HashToken(token))
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	tokenHash := security.HashToken(token)
	principal, _ := s.repo.AuthenticateSession(ctx, tokenHash)
	if err := s.repo.RevokeSession(ctx, tokenHash, "logout"); err != nil {
		return err
	}
	if principal != nil {
		s.audit.LogParticipant(ctx, principal.Participant.ID, principal.Event.ID,
			"PARTICIPANT_LOGOUT", "participant_session", "", nil)
	}
	return nil
}
