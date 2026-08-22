package auth

import (
	"context"
	"testing"
)

func TestRoleAllowedForAudience(t *testing.T) {
	t.Parallel()
	staff := []Role{{Code: "ADMIN"}}
	contestant := []Role{{Code: "CONTESTANT"}}
	both := []Role{{Code: "STAFF"}, {Code: "CONTESTANT"}}

	if !roleAllowedForAudience("", staff) || !roleAllowedForAudience("admin", staff) {
		t.Fatal("admin audience should accept ADMIN")
	}
	if roleAllowedForAudience("contestant", staff) {
		t.Fatal("contestant audience should reject ADMIN")
	}
	if !roleAllowedForAudience("contestant", contestant) {
		t.Fatal("contestant audience should accept CONTESTANT")
	}
	if roleAllowedForAudience("admin", contestant) {
		t.Fatal("admin audience should reject CONTESTANT")
	}
	if !roleAllowedForAudience("admin", both) || !roleAllowedForAudience("contestant", both) {
		t.Fatal("dual-role user can enter either portal")
	}
	if roleAllowedForAudience("unknown", staff) {
		t.Fatal("unknown audience must fail")
	}
	jury := []Role{{Code: "JURY"}}
	if !roleAllowedForAudience("jury", jury) || roleAllowedForAudience("admin", jury) || roleAllowedForAudience("contestant", jury) {
		t.Fatal("jury audience should accept only JURY")
	}
	remote := []Role{{Code: "REMOTE_JURY"}}
	if !roleAllowedForAudience("jury", remote) || roleAllowedForAudience("admin", remote) {
		t.Fatal("jury audience should accept REMOTE_JURY")
	}
	if roleAllowedForAudience("jury", staff) {
		t.Fatal("admin must not enter jury audience")
	}
}

func TestPrimaryRoleJury(t *testing.T) {
	t.Parallel()
	if got := primaryRole([]Role{{Code: "JURY"}, {Code: "CONTESTANT"}}); got != "JURY" {
		t.Fatalf("jury over contestant: %s", got)
	}
	if got := primaryRole([]Role{{Code: "ADMIN"}, {Code: "JURY"}}); got != "ADMIN" {
		t.Fatalf("admin over jury: %s", got)
	}
	if got := primaryRole([]Role{{Code: "REMOTE_JURY"}}); got != "JURY" {
		t.Fatalf("remote jury token role: %s", got)
	}
}

func TestAudienceAllowedRemoteJury(t *testing.T) {
	t.Parallel()
	s := &Service{
		juryExtra: func(_ context.Context, userID string) (bool, error) {
			return userID == "remote-1", nil
		},
	}
	if !s.audienceAllowed(context.Background(), "jury", "remote-1", nil) {
		t.Fatal("assignment-only remote jury must enter jury audience")
	}
	if s.audienceAllowed(context.Background(), "jury", "other", nil) {
		t.Fatal("unassigned user must not enter jury audience")
	}
	if s.audienceAllowed(context.Background(), "admin", "remote-1", nil) {
		t.Fatal("remote jury must not enter admin audience")
	}
	if s.tokenRole(context.Background(), "jury", "remote-1", nil) != "JURY" {
		t.Fatal("jury login must issue JURY token")
	}
}
