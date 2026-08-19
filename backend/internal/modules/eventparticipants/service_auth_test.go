package eventparticipants

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

type fakeRepo struct {
	event                   *EventRef
	eventErr                error
	nameMatches             []Participant
	nameMatchesByNormalized map[string][]Participant
	union                   *Participant
	unionErr                error
	unionByValue            map[string]*Participant
	sks                     *Participant
	sksErr                  error
	sksByValue              map[string]*Participant
	sessionHash             string
	sessionExp              time.Time
	principal               *Principal
	revokedHash             string
	created                 []*Participant
	updated                 []*Participant
	all                     []Participant
	telegramByID            map[int64]*Participant
	telegramByUsername      map[string]*Participant
	vkByID                  map[int64]*Participant
	boundTelegram           *int64
}

func (f *fakeRepo) CanManage(context.Context, string, string) (bool, error) { return true, nil }
func (f *fakeRepo) CanAccessDirections(context.Context, string, string) (bool, error) {
	return true, nil
}
func (f *fakeRepo) List(context.Context, string, ListFilter) ([]Participant, int, error) {
	return nil, 0, nil
}
func (f *fakeRepo) All(context.Context, string) ([]Participant, error) { return f.all, nil }
func (f *fakeRepo) ByID(context.Context, string, string) (*Participant, error) {
	return nil, ErrNotFound
}
func (f *fakeRepo) Create(_ context.Context, participant *Participant) (string, error) {
	copy := *participant
	f.created = append(f.created, &copy)
	return "created", nil
}
func (f *fakeRepo) Update(_ context.Context, participant *Participant) error {
	copy := *participant
	f.updated = append(f.updated, &copy)
	return nil
}
func (f *fakeRepo) SetStatus(context.Context, string, string, string) error {
	return nil
}
func (f *fakeRepo) EventBySlug(context.Context, string) (*EventRef, error) {
	return f.event, f.eventErr
}
func (f *fakeRepo) FindByNameBirth(_ context.Context, _, normalized string, _ time.Time) ([]Participant, error) {
	if f.nameMatchesByNormalized != nil {
		return f.nameMatchesByNormalized[normalized], nil
	}
	return f.nameMatches, nil
}
func (f *fakeRepo) FindByUnionCard(_ context.Context, _, value string) (*Participant, error) {
	if f.unionByValue != nil {
		participant := f.unionByValue[value]
		if participant == nil {
			return nil, ErrInvalidCredentials
		}
		return participant, nil
	}
	return f.union, f.unionErr
}
func (f *fakeRepo) FindBySKSBarcode(_ context.Context, _, value string) (*Participant, error) {
	if f.sksByValue != nil {
		participant := f.sksByValue[value]
		if participant == nil {
			return nil, ErrInvalidCredentials
		}
		return participant, nil
	}
	return f.sks, f.sksErr
}
func (f *fakeRepo) ListActiveEvents(context.Context) ([]EventRef, error) {
	if f.event == nil {
		return []EventRef{}, nil
	}
	return []EventRef{*f.event}, nil
}
func (f *fakeRepo) FindByTelegramUserID(_ context.Context, _ string, userID int64) (*Participant, error) {
	if f.telegramByID != nil {
		if p := f.telegramByID[userID]; p != nil {
			return p, nil
		}
	}
	return nil, ErrNotFound
}
func (f *fakeRepo) FindByVKUserID(_ context.Context, _ string, userID int64) (*Participant, error) {
	if f.vkByID != nil {
		if p := f.vkByID[userID]; p != nil {
			return p, nil
		}
	}
	return nil, ErrNotFound
}
func (f *fakeRepo) FindByTelegramUsername(_ context.Context, _ string, username string) (*Participant, error) {
	if f.telegramByUsername != nil {
		if p := f.telegramByUsername[strings.ToLower(username)]; p != nil {
			return p, nil
		}
	}
	return nil, ErrNotFound
}
func (f *fakeRepo) FindByVKIdentity(context.Context, string, int64, string) (*Participant, error) {
	return nil, ErrNotFound
}
func (f *fakeRepo) BindTelegram(_ context.Context, _, _ string, userID int64, _ *string) error {
	f.boundTelegram = &userID
	return nil
}
func (f *fakeRepo) BindVK(context.Context, string, string, int64, *string) error { return nil }
func (f *fakeRepo) CreateSession(
	_ context.Context, _, _, tokenHash, _, _ string, expiresAt time.Time,
) (string, error) {
	f.sessionHash = tokenHash
	f.sessionExp = expiresAt
	return "session-1", nil
}
func (f *fakeRepo) AuthenticateSession(context.Context, string) (*Principal, error) {
	if f.principal != nil {
		return f.principal, nil
	}
	return nil, ErrSessionExpired
}
func (f *fakeRepo) RevokeSession(_ context.Context, tokenHash, _ string) error {
	f.revokedHash = tokenHash
	return nil
}

func (f *fakeRepo) ListDirections(context.Context, string) ([]Direction, error) {
	return nil, nil
}
func (f *fakeRepo) CreateDirection(_ context.Context, contestID, name string) (*Direction, error) {
	return &Direction{ID: "dir-new", ContestID: contestID, Name: name}, nil
}
func (f *fakeRepo) UpdateDirection(_ context.Context, contestID, directionID, name string) (*Direction, error) {
	return &Direction{ID: directionID, ContestID: contestID, Name: name}, nil
}
func (f *fakeRepo) DeleteDirection(context.Context, string, string) error { return nil }
func (f *fakeRepo) EnsureDirection(_ context.Context, contestID, name string) (*Direction, error) {
	return &Direction{ID: "dir-" + strings.ToLower(name), ContestID: contestID, Name: name}, nil
}
func (f *fakeRepo) DirectionInContest(context.Context, string, string) (bool, error) {
	return true, nil
}

type fakeAudit struct {
	participantID string
	method        string
	action        string
}

func (*fakeAudit) Log(context.Context, string, string, string, string, map[string]any) {}
func (a *fakeAudit) LogParticipant(
	_ context.Context, participantID, _ string, action string, _ string, _ string, meta map[string]any,
) {
	a.participantID = participantID
	a.action = action
	a.method, _ = meta["method"].(string)
}

func TestLogoutRevokesHashedSessionAndWritesAudit(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{principal: &Principal{
		Participant: activeParticipant(), Event: *activeEvent(),
	}}
	audit := &fakeAudit{}
	svc := testService(repo, audit)
	if err := svc.Logout(context.Background(), "raw-session-token"); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if repo.revokedHash != security.HashToken("raw-session-token") {
		t.Fatalf("revoked hash=%q", repo.revokedHash)
	}
	if audit.action != "PARTICIPANT_LOGOUT" || audit.participantID != "participant-1" {
		t.Fatalf("audit action=%q participant=%q", audit.action, audit.participantID)
	}
}

func activeEvent() *EventRef {
	return &EventRef{ID: "contest-1", Slug: "event-2026", Name: "Event", Status: "ACTIVE", Timezone: "UTC"}
}

func activeParticipant() Participant {
	return Participant{ID: "participant-1", ContestID: "contest-1", FullName: "Иванов Иван", Status: StatusActive}
}

func testService(repo *fakeRepo, audit *fakeAudit) *Service {
	svc := NewService(repo, audit, 2*time.Hour)
	svc.now = func() time.Time { return time.Date(2026, 8, 17, 10, 0, 0, 0, time.UTC) }
	svc.newToken = func() (string, error) { return "raw-session-token", nil }
	return svc
}

func TestLoginByNameCreatesHashedSession(t *testing.T) {
	t.Parallel()
	p := activeParticipant()
	repo := &fakeRepo{event: activeEvent(), nameMatches: []Participant{p}}
	audit := &fakeAudit{}
	svc := testService(repo, audit)

	result, err := svc.LoginByName(context.Background(), "event-2026", "  ИВАНОВ  Иван ",
		time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), ClientInfo{IP: "192.0.2.1"})
	if err != nil {
		t.Fatalf("LoginByName: %v", err)
	}
	if result.Token != "raw-session-token" {
		t.Fatalf("token = %q", result.Token)
	}
	if repo.sessionHash != security.HashToken("raw-session-token") || repo.sessionHash == result.Token {
		t.Fatalf("session hash was not stored safely: %q", repo.sessionHash)
	}
	wantExp := svc.now().Add(2 * time.Hour)
	if !repo.sessionExp.Equal(wantExp) {
		t.Fatalf("expires = %v, want %v", repo.sessionExp, wantExp)
	}
	if audit.participantID != p.ID || audit.method != "fio_birth_date" {
		t.Fatalf("audit = participant %q method %q", audit.participantID, audit.method)
	}
}

func TestLoginByNameRejectsAmbiguousIdentity(t *testing.T) {
	t.Parallel()
	p := activeParticipant()
	repo := &fakeRepo{event: activeEvent(), nameMatches: []Participant{p, p}}
	svc := testService(repo, &fakeAudit{})
	_, err := svc.LoginByName(context.Background(), "event-2026", "Иванов Иван",
		time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC), ClientInfo{})
	if !errors.Is(err, ErrAmbiguousIdentity) {
		t.Fatalf("error = %v, want ErrAmbiguousIdentity", err)
	}
}

func TestLoginByIdentifiers(t *testing.T) {
	t.Parallel()
	for name, setup := range map[string]func(*fakeRepo, *Participant){
		"union card":  func(repo *fakeRepo, p *Participant) { repo.union = p },
		"sks barcode": func(repo *fakeRepo, p *Participant) { repo.sks = p },
	} {
		name, setup := name, setup
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			p := activeParticipant()
			repo := &fakeRepo{event: activeEvent()}
			setup(repo, &p)
			svc := testService(repo, &fakeAudit{})
			var err error
			if name == "union card" {
				_, err = svc.LoginByUnionCard(context.Background(), "event-2026", " 001 ", ClientInfo{})
			} else {
				_, err = svc.LoginBySKSBarcode(context.Background(), "event-2026", " code ", ClientInfo{})
			}
			if err != nil {
				t.Fatalf("login: %v", err)
			}
		})
	}
}

func TestLoginRejectsBlockedParticipantAndUnavailableEvent(t *testing.T) {
	t.Parallel()
	blocked := activeParticipant()
	blocked.Status = StatusBlocked
	repo := &fakeRepo{event: activeEvent(), union: &blocked}
	svc := testService(repo, &fakeAudit{})
	if _, err := svc.LoginByUnionCard(context.Background(), "event-2026", "001", ClientInfo{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("blocked error = %v", err)
	}

	repo.event = &EventRef{ID: "contest-1", Slug: "event-2026", Status: "FINISHED"}
	if _, err := svc.LoginByUnionCard(context.Background(), "event-2026", "001", ClientInfo{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("finished event error = %v", err)
	}
}

func TestLoginDoesNotHideRepositoryFailure(t *testing.T) {
	t.Parallel()
	want := errors.New("database unavailable")
	svc := testService(&fakeRepo{eventErr: want}, &fakeAudit{})
	if _, err := svc.LoginByUnionCard(context.Background(), "event-2026", "001", ClientInfo{}); !errors.Is(err, want) {
		t.Fatalf("error = %v, want repository failure", err)
	}
}
