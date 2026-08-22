package evaluation

import "time"

func phaseElapsed(now time.Time, started, paused *time.Time, acc float64) *float64 {
	if started == nil {
		return nil
	}
	end := now
	if paused != nil {
		end = *paused
	}
	elapsed := end.Sub(*started).Seconds() - acc
	if elapsed < 0 {
		elapsed = 0
	}
	return &elapsed
}

func timerRemaining(now time.Time, duration *int, started, paused *time.Time, acc float64) *float64 {
	if duration == nil || started == nil {
		return nil
	}
	end := now
	if paused != nil {
		end = *paused
	}
	elapsed := end.Sub(*started).Seconds() - acc
	if elapsed < 0 {
		elapsed = 0
	}
	rem := float64(*duration) - elapsed
	return &rem
}

func sessionLocked(state string) bool {
	return state != "" && state != StateNotStarted && state != StatePreparing
}

func drawLocked(state string) bool {
	return state != StateNotStarted && state != StatePreparing
}
