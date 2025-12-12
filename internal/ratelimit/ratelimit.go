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

var (
	rateLimits = make(map[string]*rateInfo)
	rateMu     sync.Mutex
)

const (
	exportLimit  = 5
	exportWindow = 1 * time.Hour
)

func Export(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "invalid ip", 400)
			return
		}

		now := time.Now()

		rateMu.Lock()
		info, exists := rateLimits[ip]

		if !exists || now.After(info.ResetTime) {
			info = &rateInfo{
				Count:     0,
				ResetTime: now.Add(exportWindow),
			}
			rateLimits[ip] = info
		}

		if info.Count >= exportLimit {
			rateMu.Unlock()
			http.Error(w, "rate limit exceeded", 429)
			return
		}

		info.Count++
		rateMu.Unlock()

		next(w, r)
	}
}
