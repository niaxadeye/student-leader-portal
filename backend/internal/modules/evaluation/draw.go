package evaluation

import (
	"crypto/rand"
	"math/big"
)

func shuffleStrings(ids []string) error {
	for i := len(ids) - 1; i > 0; i-- {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(n.Int64())
		ids[i], ids[j] = ids[j], ids[i]
	}
	return nil
}

// mergeDrawOrder принимает запрошенный порядок и дописывает в конец тех,
// кого не перечислили (например, новых конкурсантов после жеребьёвки).
func mergeDrawOrder(currentIDs, requested []string) ([]string, error) {
	allowed := make(map[string]struct{}, len(currentIDs))
	for _, id := range currentIDs {
		allowed[id] = struct{}{}
	}
	seen := make(map[string]struct{}, len(requested))
	out := make([]string, 0, len(currentIDs))
	for _, id := range requested {
		if _, ok := allowed[id]; !ok {
			return nil, ErrValidation
		}
		if _, dup := seen[id]; dup {
			return nil, ErrValidation
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range currentIDs {
		if _, ok := seen[id]; !ok {
			out = append(out, id)
		}
	}
	return out, nil
}
