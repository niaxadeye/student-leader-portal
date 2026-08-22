package contests

import (
	"errors"
)

// ErrLoginConflict — логин уже занят чужим или привилегированным аккаунтом.
// HTTP 409 без идентификатора: нельзя светить UUID для последующего захвата.
var ErrLoginConflict = errors.New("login already taken")

type existingAccount struct {
	ID        string
	CreatedBy *string
	Roles     []string
}

func privilegedRoles(roles []string) bool {
	for _, r := range roles {
		if r != "" && r != "CONTESTANT" {
			return true
		}
	}
	return false
}

// canAttachExistingContestant разрешает привязать уже существующий логин к конкурсу,
// не меняя пароль/профиль. Чужой tenant и привилегированные роли — конфликт.
func canAttachExistingContestant(actor Actor, acc existingAccount) error {
	if privilegedRoles(acc.Roles) {
		return ErrLoginConflict
	}
	if actor.IsMega {
		return nil
	}
	if acc.CreatedBy != nil && *acc.CreatedBy == actor.UserID {
		return nil
	}
	return ErrLoginConflict
}
