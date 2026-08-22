package evaluation

import (
	"sync"
	"time"
)

type LiveEvent struct {
	Type     string `json:"type"`
	Revision int    `json:"revision"`
	State    string `json:"state,omitempty"`
}

type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan LiveEvent]struct{}
	seen map[string]map[string]time.Time
}

func NewHub() *Hub {
	return &Hub{
		subs: map[string]map[chan LiveEvent]struct{}{},
		seen: map[string]map[string]time.Time{},
	}
}

func (h *Hub) Subscribe(challengeID string) (<-chan LiveEvent, func()) {
	ch := make(chan LiveEvent, 8)
	h.mu.Lock()
	if h.subs[challengeID] == nil {
		h.subs[challengeID] = map[chan LiveEvent]struct{}{}
	}
	h.subs[challengeID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if m := h.subs[challengeID]; m != nil {
			delete(m, ch)
			if len(m) == 0 {
				delete(h.subs, challengeID)
			}
		}
		close(ch)
	}
}

func (h *Hub) Publish(challengeID string, ev LiveEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[challengeID] {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (h *Hub) Touch(challengeID, userID string) {
	if challengeID == "" || userID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.seen[challengeID] == nil {
		h.seen[challengeID] = map[string]time.Time{}
	}
	h.seen[challengeID][userID] = time.Now()
}

func (h *Hub) Online(challengeID string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.seen[challengeID]
	if m == nil {
		return 0
	}
	cutoff := time.Now().Add(-45 * time.Second)
	n := 0
	for id, t := range m {
		if t.Before(cutoff) {
			delete(m, id)
			continue
		}
		n++
	}
	return n
}
