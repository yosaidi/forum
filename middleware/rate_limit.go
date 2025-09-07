package middleware

import (
	"net/http"
	"sync"
	"time"

	"forum/utils"
)

type visitor struct {
	requests int
	lastSeen time.Time
}

var (
	visitors    = make(map[string]*visitor)
	mu          sync.Mutex
	maxRequests = 5
	window      = 10 * time.Second
)

// cleanup old visitors every minute
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > window {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

// getVisitor retrieves the visitor for a given IP address, creating one if it doesn't exist.
func getVisitor(ip string) *visitor {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		v = &visitor{requests: 1, lastSeen: time.Now()}
		visitors[ip] = v
		return v
	}
	if time.Since(v.lastSeen) > window {
		v.requests = 1
		v.lastSeen = time.Now()
		return v
	}

	v.requests++
	v.lastSeen = time.Now()
	return v
}

// RateLimit is a placeholder for rate limiting middleware implementation.
func RateLimit(next http.Handler) http.Handler {
	go cleanupVisitors() // run cleaner once in background

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		v := getVisitor(ip)

		if v.requests > maxRequests {
			utils.TooManyRequests(w, "Too many requests.Please slow down.")
			return
		}

		next.ServeHTTP(w, r)
	})
}
