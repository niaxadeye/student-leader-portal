package useradmin

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type recAudit struct {
	events []string
}

func (a *recAudit) Log(_ context.Context, _, action, _, entityID string, _ map[string]any) {
	a.events = append(a.events, action+":"+entityID)
}

func (a *recAudit) has(prefix string) bool {
	for _, e := range a.events {
		if strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}

type fakeStore struct {
	targets     map[string]*AccessTarget
	users       map[string]*User
	owns        map[string]bool // actorID+"/"+userID
	contestOwn  map[string]bool // contestID+"/"+userID
	passwordOf  map[string]string
	statusOf    map[string]string
	revoked     []string
	assigned    []string
	updated     []string
	mutateCalls int
}

func (f *fakeStore) List(context.Context, ListFilter) ([]User, int, error) { return nil, 0, nil }
func (f *fakeStore) ByID(_ context.Context, id string) (*User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}
func (f *fakeStore) Create(context.Context, NewUser, string, string, string, string) (string, error) {
	return "", nil
}
func (f *fakeStore) UpdateProfile(_ context.Context, id, _ string, _, _ *string) error {
	f.mutateCalls++
	f.updated = append(f.updated, id)
	return nil
}
func (f *fakeStore) AssignRole(_ context.Context, userID, role, _, _, _ string) error {
	f.mutateCalls++
	f.assigned = append(f.assigned, userID+":"+role)
	return nil
}
func (f *fakeStore) RemoveRole(_ context.Context, userID, role, _, _ string) error {
	f.mutateCalls++
	f.assigned = append(f.assigned, "rm:"+userID+":"+role)
	return nil
}
func (f *fakeStore) AccessTarget(_ context.Context, userID string) (*AccessTarget, error) {
	t, ok := f.targets[userID]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *t
	return &cp, nil
}
func (f *fakeStore) OwnsContestant(_ context.Context, actorID, userID string) (bool, error) {
	return f.owns[actorID+"/"+userID], nil
}
func (f *fakeStore) ContestOwnedBy(_ context.Context, contestID, userID string) (bool, error) {
	if v, ok := f.contestOwn[contestID+"/"+userID]; ok {
		return v, nil
	}
	return false, nil
}
func (f *fakeStore) SetPassword(_ context.Context, userID, hash string) error {
	f.mutateCalls++
	if f.passwordOf == nil {
		f.passwordOf = map[string]string{}
	}
	f.passwordOf[userID] = hash
	return nil
}
func (f *fakeStore) SetStatus(_ context.Context, userID, status string) error {
	f.mutateCalls++
	if f.statusOf == nil {
		f.statusOf = map[string]string{}
	}
	f.statusOf[userID] = status
	return nil
}
func (f *fakeStore) RevokeSessions(_ context.Context, userID, _ string) error {
	f.revoked = append(f.revoked, userID)
	return nil
}

func seedStore() *fakeStore {
	return &fakeStore{
		targets: map[string]*AccessTarget{
			"own-c":     {ID: "own-c", CreatedBy: ptr("super-a"), Roles: []string{"CONTESTANT"}},
			"own-admin": {ID: "own-admin", CreatedBy: ptr("super-a"), Roles: []string{"ADMIN"}},
			"foreign-c": {ID: "foreign-c", CreatedBy: ptr("super-b"), Roles: []string{"CONTESTANT"}},
			"foreign-a": {ID: "foreign-a", CreatedBy: ptr("super-b"), Roles: []string{"ADMIN"}},
			"super-b":   {ID: "super-b", CreatedBy: ptr("mega"), Roles: []string{"SUPER_ADMIN"}},
			"mega":      {ID: "mega", CreatedBy: nil, Roles: []string{"MEGA_ADMIN"}},
			"shared-c":  {ID: "shared-c", CreatedBy: ptr("super-b"), Roles: []string{"CONTESTANT"}},
		},
		users: map[string]*User{
			"own-c": {ID: "own-c", Login: "own-c"},
		},
		owns: map[string]bool{
			"super-a/shared-c": true,
		},
		contestOwn: map[string]bool{
			"contest-a/super-a": true,
			"contest-b/super-b": true,
		},
	}
}

func TestResetPasswordForbiddenDoesNotMutate(t *testing.T) {
	t.Parallel()
	store := seedStore()
	audit := &recAudit{}
	svc := newService(store, audit)
	admin := Actor{UserID: "admin", Role: "ADMIN"}
	superA := Actor{UserID: "super-a", Role: "SUPER_ADMIN"}

	if _, err := svc.ResetPassword(context.Background(), admin, "own-c"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("ADMIN reset: %v", err)
	}
	if _, err := svc.ResetPassword(context.Background(), superA, "foreign-a"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("SUPER A foreign: %v", err)
	}
	if _, err := svc.ResetPassword(context.Background(), superA, "super-b"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("SUPER A other SUPER: %v", err)
	}
	if _, err := svc.ResetPassword(context.Background(), superA, "mega"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("SUPER A MEGA: %v", err)
	}
	if store.mutateCalls != 0 {
		t.Fatalf("forbidden reset mutated store: %d", store.mutateCalls)
	}
	if !audit.has("USER_ACCESS_DENIED:") {
		t.Fatal("expected access-denied audit")
	}
}

func TestResetPasswordAllowedAndRevokes(t *testing.T) {
	t.Parallel()
	store := seedStore()
	svc := newService(store, &recAudit{})
	superA := Actor{UserID: "super-a", Role: "SUPER_ADMIN"}
	mega := Actor{UserID: "mega", Role: "MEGA_ADMIN"}

	if _, err := svc.ResetPassword(context.Background(), superA, "own-c"); err != nil {
		t.Fatalf("own contestant: %v", err)
	}
	if _, err := svc.ResetPassword(context.Background(), superA, "shared-c"); err != nil {
		t.Fatalf("contestant in owned contest: %v", err)
	}
	if _, err := svc.ResetPassword(context.Background(), mega, "foreign-a"); err != nil {
		t.Fatalf("mega foreign: %v", err)
	}
	if store.passwordOf["own-c"] == "" || store.passwordOf["shared-c"] == "" || store.passwordOf["foreign-a"] == "" {
		t.Fatalf("passwords not set: %+v", store.passwordOf)
	}
	if len(store.revoked) < 3 {
		t.Fatalf("sessions not revoked: %v", store.revoked)
	}
}

func TestSetStatusForbiddenAndMegaFreeze(t *testing.T) {
	t.Parallel()
	store := seedStore()
	svc := newService(store, &recAudit{})
	superA := Actor{UserID: "super-a", Role: "SUPER_ADMIN"}
	if err := svc.SetStatus(context.Background(), superA, "super-b", "BLOCKED"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("super cannot freeze other super: %v", err)
	}
	if store.mutateCalls != 0 {
		t.Fatal("forbidden block mutated")
	}
	mega := Actor{UserID: "mega", Role: "MEGA_ADMIN"}
	if err := svc.SetStatus(context.Background(), mega, "super-b", "BLOCKED"); err != nil {
		t.Fatalf("mega freeze: %v", err)
	}
	if store.statusOf["super-b"] != "BLOCKED" {
		t.Fatalf("status=%q", store.statusOf["super-b"])
	}
}

func TestGetAndUpdateScoped(t *testing.T) {
	t.Parallel()
	store := seedStore()
	svc := newService(store, &recAudit{})
	superA := Actor{UserID: "super-a", Role: "SUPER_ADMIN"}
	if _, err := svc.Get(context.Background(), superA, "foreign-a"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("get foreign: %v", err)
	}
	if _, err := svc.Update(context.Background(), superA, "foreign-a", "Name", "", ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("update foreign: %v", err)
	}
	if _, err := svc.Get(context.Background(), superA, "own-c"); err != nil {
		t.Fatalf("get own: %v", err)
	}
}

func TestAssignRoleRejectsForeignAndGlobalAdmin(t *testing.T) {
	t.Parallel()
	store := seedStore()
	svc := newService(store, &recAudit{})
	superA := Actor{UserID: "super-a", Role: "SUPER_ADMIN"}

	err := svc.AssignRole(context.Background(), superA, "shared-c", AssignRoleInput{
		Role: "ADMIN", ScopeType: "CONTEST", ScopeID: "contest-a", AccessLevel: "EDIT",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("grant to foreign contestant in own contest: %v", err)
	}
	err = svc.AssignRole(context.Background(), superA, "own-admin", AssignRoleInput{Role: "ADMIN", ScopeType: "GLOBAL"})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("global ADMIN: %v", err)
	}
	if store.mutateCalls != 0 {
		t.Fatalf("rejected assign mutated: %d", store.mutateCalls)
	}

	if err := svc.AssignRole(context.Background(), superA, "own-c", AssignRoleInput{
		Role: "ADMIN", ScopeType: "CONTEST", ScopeID: "contest-a", AccessLevel: "VIEW",
	}); err != nil {
		t.Fatalf("grant own: %v", err)
	}
	if len(store.revoked) == 0 {
		t.Fatal("role change must revoke sessions")
	}
}

func TestCreateGlobalAdminRejected(t *testing.T) {
	t.Parallel()
	store := seedStore()
	svc := newService(store, &recAudit{})
	_, err := svc.Create(context.Background(), Actor{UserID: "super-a", Role: "SUPER_ADMIN"}, CreateInput{
		Login: "a1", FullName: "A", Role: "ADMIN",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("create global ADMIN: %v", err)
	}
}
