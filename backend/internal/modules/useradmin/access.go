package useradmin

// ManageAction — вид админ-операции над целевым пользователем.
// Граница безопасности — сервисный слой, не роутер и не UI.
type ManageAction string

const (
	ActionView   ManageAction = "view"
	ActionUpdate ManageAction = "update"
	ActionReset  ManageAction = "reset"
	ActionStatus ManageAction = "status"
	ActionGrant  ManageAction = "grant"
)

// AccessTarget — снимок цели для CanManageUser (роли + владение).
type AccessTarget struct {
	ID        string
	CreatedBy *string
	Roles     []string
}

func roleRank(role string) int {
	switch role {
	case "MEGA_ADMIN":
		return 5
	case "SUPER_ADMIN":
		return 4
	case "ADMIN":
		return 3
	case "STAFF":
		return 2
	case "CONTESTANT":
		return 1
	default:
		return 0
	}
}

// HighestRole выбирает основную роль по иерархии JWT.
func HighestRole(roles []string) string {
	best, rank := "CONTESTANT", 0
	for _, r := range roles {
		if roleRank(r) > rank {
			best, rank = r, roleRank(r)
		}
	}
	return best
}

func contestantOnly(roles []string) bool {
	if len(roles) == 0 {
		return true
	}
	for _, r := range roles {
		if r != "CONTESTANT" {
			return false
		}
	}
	return true
}

func ownedBy(t AccessTarget, actorID string) bool {
	return t.CreatedBy != nil && *t.CreatedBy == actorID
}

// CanManageUser — единый guard админ-операций над пользователем (§3.3, O6).
//
//   - MEGA_ADMIN — глобально;
//   - SUPER_ADMIN — только своя цепочка created_by; для view/update/reset/status
//     дополнительно свои конкурсанты (CONTESTANT-only в принадлежащем конкурсе);
//   - назначение ролей (grant) — только created_by, без «чужой конкурсант в моём событии»;
//   - SUPER не трогает MEGA/другого SUPER;
//   - ADMIN/STAFF/CONTESTANT — всегда запрещено.
func CanManageUser(actor Actor, t AccessTarget, inActorContest bool, action ManageAction) error {
	if actor.UserID == "" {
		return ErrForbidden
	}
	if actor.IsMega() {
		return nil
	}
	if !actor.IsSuper() {
		return ErrForbidden
	}
	if roleRank(HighestRole(t.Roles)) >= roleRank("SUPER_ADMIN") {
		return ErrForbidden
	}
	if ownedBy(t, actor.UserID) {
		return nil
	}
	if action != ActionGrant && inActorContest && contestantOnly(t.Roles) {
		return nil
	}
	return ErrForbidden
}
