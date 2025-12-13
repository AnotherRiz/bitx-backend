package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type rateInfo struct {
	Count     int
	ResetTime time.Time
}

type Limiter struct {
	Limit  int
	Window time.Duration
}

var (
	store = make(map[string]*rateInfo)
	mu    sync.Mutex
)

// core limiter
func limit(l Limiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "invalid ip", 400)
			return
		}

		key := ip + ":" + r.URL.Path
		now := time.Now()

		mu.Lock()
		info, exists := store[key]

		if !exists || now.After(info.ResetTime) {
			info = &rateInfo{
				Count:     0,
				ResetTime: now.Add(l.Window),
			}
			store[key] = info
		}

		if info.Count >= l.Limit {
			mu.Unlock()
			http.Error(w, "rate limit exceeded", 429)
			return
		}

		info.Count++
		mu.Unlock()

		next(w, r)
	}
}

// ====== public wrappers ======

var ExportLimiter = Limiter{
	Limit:  5,
	Window: 1 * time.Hour,
}

var ImportLimiter = Limiter{
	Limit:  10,
	Window: 5 * time.Minute,
}

func Export(next http.HandlerFunc) http.HandlerFunc {
	return limit(ExportLimiter, next)
}

func Import(next http.HandlerFunc) http.HandlerFunc {
	return limit(ImportLimiter, next)
}
