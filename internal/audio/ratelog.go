package audio

import (
	"log/slog"
	"sync"
	"time"
)

// rateLimitedLog emits at most one message per key per interval.
type rateLimitedLog struct {
	mu       sync.Mutex
	last     map[string]time.Time
	interval time.Duration
}

func newRateLimitedLog(interval time.Duration) *rateLimitedLog {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &rateLimitedLog{last: make(map[string]time.Time), interval: interval}
}

func (r *rateLimitedLog) allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if t, ok := r.last[key]; ok && now.Sub(t) < r.interval {
		return false
	}
	r.last[key] = now
	return true
}

func (r *rateLimitedLog) Info(log *slog.Logger, key, msg string, args ...any) {
	if log == nil || !r.allow(key) {
		return
	}
	log.Info(msg, args...)
}

func (r *rateLimitedLog) Warn(log *slog.Logger, key, msg string, args ...any) {
	if log == nil || !r.allow(key) {
		return
	}
	log.Warn(msg, args...)
}
