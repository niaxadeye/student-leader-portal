package eventpermissions

import (
	"context"
	"errors"
	"slices"
	"testing"
)

type fakeRepo struct {
	owner     string
	users     map[string]*userRef
	grants    map[string][]Grant
	byContest map[string][]Assignment
	replaced  []string
	cleared   []string
}

func (f *fakeRepo) ListByUser(_ context.Context, userID string) ([]Grant, error) {
	return f.grants[userID], nil
}
func (f *fakeRepo) ListByContest(_ context.Context, contestID string) ([]Assignment, error) {
	return f.byContest[contestID], nil
}
func (f *fakeRepo) Replace(_ context.Context, contestID, userID, _ string, permissions []string) error {
	f.replaced = append(f.replaced, contestID+":"+userID)
	f.grants[userID] = []Grant{{ContestID: contestID, Permissions: permissions}}
	return nil
}
func (f *fakeRepo) Clear(_ context.Context, contestID, userID string) error {
	f.cleared = append(f.cleared, contestID+":"+userID)
	return nil
}
func (f *fakeRepo) ContestOwner(_ context.Context, contestID string) (string, error) {
	if contestID == "missing" {
		return "", ErrNotFound
	}
	if f.owner == "" {
		return "", ErrNotFound
	}
	return f.owner, nil
}
func (f *fakeRepo) UserRef(_ context.Context, userID string) (*userRef, error) {
	u, ok := f.users[userID]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func createdBy(id string) *string { return &id }

func TestReplaceStaffPermissions(t *testing.T) {
	t.Parallel()
	owner := "super-a"
	staff := "staff-1"
	repo := &fakeRepo{
		owner:  owner,
		users:  map[string]*userRef{staff: {CreatedBy: createdBy(owner), Roles: []string{"STAFF"}}},
		grants: map[string][]Grant{},
	}
	svc := NewService(repo, nil)
	actor := Actor{UserID: owner, Role: "SUPER_ADMIN"}

	if err := svc.Replace(context.Background(), actor, "contest-a", staff, []string{AttendanceScan}); err != nil {
		t.Fatalf("owner grant: %v", err)
	}
	if err := svc.Replace(context.Background(), actor, "contest-a", staff, []string{"event.unknown"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown permission: got %v", err)
	}

	other := Actor{UserID: "super-b", Role: "SUPER_ADMIN"}
	if err := svc.Replace(context.Background(), other, "contest-a", staff, []string{AttendanceScan}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("foreign owner: got %v", err)
	}

	admin := Actor{UserID: owner, Role: "ADMIN"}
	if err := svc.Replace(context.Background(), admin, "contest-a", staff, []string{AttendanceScan}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("admin grant: got %v", err)
	}

	staffActor := Actor{UserID: staff, Role: "STAFF"}
	if err := svc.Replace(context.Background(), staffActor, "contest-a", staff, []string{AttendanceScan}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("staff self-grant: got %v", err)
	}
}

func TestReplaceRejectsPrivilegedTargetAndForeignStaff(t *testing.T) {
	t.Parallel()
	owner := "super-a"
	repo := &fakeRepo{
		owner: owner,
		users: map[string]*userRef{
			"mega":    {CreatedBy: nil, Roles: []string{"MEGA_ADMIN"}},
			"super-b": {CreatedBy: createdBy("mega"), Roles: []string{"SUPER_ADMIN"}},
			"admin":   {CreatedBy: createdBy(owner), Roles: []string{"ADMIN"}},
			"staff-b": {CreatedBy: createdBy("super-b"), Roles: []string{"STAFF"}},
		},
		grants: map[string][]Grant{},
	}
	svc := NewService(repo, nil)
	actor := Actor{UserID: owner, Role: "SUPER_ADMIN"}

	for _, target := range []string{"mega", "super-b", "admin", "staff-b"} {
		if err := svc.Replace(context.Background(), actor, "contest-a", target, []string{AttendanceScan}); !errors.Is(err, ErrForbidden) {
			t.Fatalf("%s: got %v", target, err)
		}
	}
}

func TestMegaCanGrantAnyStaff(t *testing.T) {
	t.Parallel()
	repo := &fakeRepo{
		owner:  "super-b",
		users:  map[string]*userRef{"staff-b": {CreatedBy: createdBy("super-b"), Roles: []string{"STAFF"}}},
		grants: map[string][]Grant{},
	}
	svc := NewService(repo, nil)
	if err := svc.Replace(context.Background(), Actor{UserID: "mega", Role: "MEGA_ADMIN"}, "contest-b", "staff-b", []string{MerchOrdersManage}); err != nil {
		t.Fatalf("mega grant: %v", err)
	}
}

func TestClearStaffPermissions(t *testing.T) {
	t.Parallel()
	owner := "super-a"
	repo := &fakeRepo{
		owner:  owner,
		users:  map[string]*userRef{"staff-1": {CreatedBy: createdBy(owner), Roles: []string{"STAFF"}}},
		grants: map[string][]Grant{},
	}
	svc := NewService(repo, nil)
	if err := svc.Replace(context.Background(), Actor{UserID: owner, Role: "SUPER_ADMIN"}, "contest-a", "staff-1", nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if !slices.Contains(repo.cleared, "contest-a:staff-1") {
		t.Fatalf("clear not called: %v", repo.cleared)
	}
}

func TestListStaffIsolation(t *testing.T) {
	t.Parallel()
	owner := "super-a"
	repo := &fakeRepo{
		owner: owner,
		users: map[string]*userRef{
			"staff-1": {CreatedBy: createdBy(owner), Roles: []string{"STAFF"}},
			"staff-b": {CreatedBy: createdBy("super-b"), Roles: []string{"STAFF"}},
		},
		grants: map[string][]Grant{"staff-1": {{ContestID: "contest-a", Permissions: []string{AttendanceScan}}}},
	}
	svc := NewService(repo, nil)
	actor := Actor{UserID: owner, Role: "SUPER_ADMIN"}
	list, err := svc.ListByUser(context.Background(), actor, "staff-1")
	if err != nil || len(list) != 1 {
		t.Fatalf("own staff: %v %#v", err, list)
	}
	if _, err := svc.ListByUser(context.Background(), actor, "staff-b"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("foreign staff list: got %v", err)
	}
	if _, err := svc.ListByContest(context.Background(), Actor{UserID: "super-b", Role: "SUPER_ADMIN"}, "contest-a"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("foreign contest staff: got %v", err)
	}
}

func TestAllPermissionsAreKnown(t *testing.T) {
	t.Parallel()
	for _, perm := range All() {
		if !IsKnown(perm) {
			t.Fatalf("All() contains unknown %q", perm)
		}
	}
}
