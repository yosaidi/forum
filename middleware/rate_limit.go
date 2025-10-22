package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"forum/utils"
)

type visitor struct {
	requests map[string]int       // key: endpoint category, value: request count
	lastSeen map[string]time.Time // key: endpoint category, value: last request time
	mu       sync.RWMutex
}

type RateLimitConfig struct {
	MaxRequests int
	Window      time.Duration
}

var (
	visitors = make(map[string]*visitor)
	globalMu sync.RWMutex

	rateLimits = map[string]RateLimitConfig{
		"auth":       {MaxRequests: 10, Window: 10 * time.Minute},
		"posts":      {MaxRequests: 100, Window: time.Hour},
		"comments":   {MaxRequests: 80, Window: time.Hour},
		"users":      {MaxRequests: 30, Window: 10 * time.Minute},
		"categories": {MaxRequests: 50, Window: time.Hour},
		"default":    {MaxRequests: 60, Window: time.Minute},
	}

	cleanupOnce sync.Once
	maxVisitors = 10000
)

// startCleanup ensures the cleanup goroutine runs only once
func startCleanup() {
	cleanupOnce.Do(func() {
		go cleanupVisitors()
	})
}

// cleanupVisitors removes stale visitor records periodically
func cleanupVisitors() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		globalMu.Lock()
		now := time.Now()
		toDelete := make([]string, 0)

		for ip, v := range visitors {
			v.mu.RLock()
			allStale := true

			// Check if all categories for this visitor are outside their windows
			for category, lastSeen := range v.lastSeen {
				config := getRateLimitConfigUnsafe(category)
				if now.Sub(lastSeen) <= config.Window {
					allStale = false
					break
				}
			}

			v.mu.RUnlock()

			if allStale {
				toDelete = append(toDelete, ip)
			} else {
				// Clean up individual stale categories
				v.mu.Lock()
				for category, lastSeen := range v.lastSeen {
					config := getRateLimitConfigUnsafe(category)
					if now.Sub(lastSeen) > config.Window {
						delete(v.requests, category)
						delete(v.lastSeen, category)
					}
				}
				v.mu.Unlock()
			}
		}

		// Delete stale visitors
		for _, ip := range toDelete {
			delete(visitors, ip)
		}
		globalMu.Unlock()
	}
}

// extractIP extracts the IP address from RemoteAddr
func extractIP(remoteAddr string) string {
	if ip, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return ip
	}
	return remoteAddr
}

// getVisitor retrieves or creates a visitor for a given IP address
func getVisitor(ip string) *visitor {
	globalMu.Lock()
	defer globalMu.Unlock()

	// Return existing visitor
	if v, exists := visitors[ip]; exists {
		return v
	}

	// Evict oldest visitor if at capacity
	if len(visitors) >= maxVisitors {
		var oldestIP string
		oldestTime := time.Now()

		for visitorIP, visitorData := range visitors {
			visitorData.mu.RLock()
			for _, t := range visitorData.lastSeen {
				if t.Before(oldestTime) {
					oldestTime = t
					oldestIP = visitorIP
				}
			}
			visitorData.mu.RUnlock()
		}

		if oldestIP != "" {
			delete(visitors, oldestIP)
		}
	}

	// Create new visitor
	v := &visitor{
		requests: make(map[string]int),
		lastSeen: make(map[string]time.Time),
	}
	visitors[ip] = v

	return v
}

// getRateLimitConfig gets the rate limit configuration for a category (thread-safe)
func getRateLimitConfig(category string) RateLimitConfig {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return getRateLimitConfigUnsafe(category)
}

// getRateLimitConfigUnsafe gets config without acquiring lock (caller must hold lock)
func getRateLimitConfigUnsafe(category string) RateLimitConfig {
	if config, exists := rateLimits[category]; exists {
		return config
	}
	return rateLimits["default"]
}

// determineCategory extracts the endpoint category from the request path
func determineCategory(path string) string {
	path = strings.Trim(path, "/")

	if path == "" {
		return "default"
	}

	// Get first segment as category
	segments := strings.SplitN(path, "/", 2)
	return segments[0]
}

// RateLimit is the general rate limiter middleware
func RateLimit(next http.Handler) http.Handler {
	startCleanup()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract IP
		ip := extractIP(r.RemoteAddr)

		// Determine category from path
		path := strings.TrimPrefix(r.URL.Path, "/")
		category := "default"

		// Strip /api prefix if present
		path = strings.TrimPrefix(path, "api/")

		category = determineCategory(path)
		if category == "" {
			category = "default"
		}

		// Get visitor and config
		v := getVisitor(ip)
		cfg := getRateLimitConfig(category)
		now := time.Now()

		// Apply rate limiting logic
		v.mu.Lock()

		lastSeen, exists := v.lastSeen[category]
		if !exists || now.Sub(lastSeen) > cfg.Window {
			// Reset counter if window expired
			v.requests[category] = 0
		}

		v.requests[category]++
		v.lastSeen[category] = now
		reqCount := v.requests[category]

		v.mu.Unlock()

		// Set rate limit headers
		setRateLimitHeaders(w, cfg, reqCount, now)

		// Check if limit exceeded (>= not > to properly enforce MaxRequests)
		if reqCount > cfg.MaxRequests {
			utils.TooManyRequests(w,
				fmt.Sprintf("Rate limit exceeded for %s: %d requests allowed per %v",
					category, cfg.MaxRequests, cfg.Window))
			return
		}

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// setRateLimitHeaders sets standard X-RateLimit-* headers
func setRateLimitHeaders(w http.ResponseWriter, cfg RateLimitConfig, count int, lastSeen time.Time) {
	remaining := cfg.MaxRequests - count
	if remaining < 0 {
		remaining = 0
	}

	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.MaxRequests))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(lastSeen.Add(cfg.Window).Unix(), 10))
}

// SetRateLimit allows dynamic configuration of rate limits
func SetRateLimit(category string, maxRequests int, window time.Duration) {
	globalMu.Lock()
	defer globalMu.Unlock()
	rateLimits[category] = RateLimitConfig{
		MaxRequests: maxRequests,
		Window:      window,
	}
}

// GetRateLimits returns a copy of current rate limit configurations
func GetRateLimits() map[string]RateLimitConfig {
	globalMu.RLock()
	defer globalMu.RUnlock()

	result := make(map[string]RateLimitConfig, len(rateLimits))
	for k, v := range rateLimits {
		result[k] = v
	}
	return result
}

// GetVisitorStats returns statistics about current visitors
func GetVisitorStats() map[string]interface{} {
	globalMu.RLock()
	defer globalMu.RUnlock()

	stats := map[string]interface{}{
		"total_visitors": len(visitors),
		"max_visitors":   maxVisitors,
	}

	categoryStats := make(map[string]int)
	for _, v := range visitors {
		v.mu.RLock()
		for category := range v.requests {
			categoryStats[category]++
		}
		v.mu.RUnlock()
	}
	stats["categories"] = categoryStats

	return stats
}

// SetMaxVisitors configures the maximum number of tracked visitors
func SetMaxVisitors(max int) {
	globalMu.Lock()
	defer globalMu.Unlock()
	maxVisitors = max
}

// ResetVisitor clears rate limit data for a specific IP (useful for testing)
func ResetVisitor(ip string) {
	globalMu.Lock()
	defer globalMu.Unlock()
	delete(visitors, ip)
}

// ResetAllVisitors clears all visitor data (useful for testing)
func ResetAllVisitors() {
	globalMu.Lock()
	defer globalMu.Unlock()
	visitors = make(map[string]*visitor)
}

// PrintRateLimitConfig displays current rate limit configuration
func PrintRateLimitConfig() {
	limits := GetRateLimits()
	fmt.Println("\n=== Rate Limit Configuration ===")

	// Print in consistent order
	categories := []string{"auth", "posts", "comments", "users", "categories", "default"}
	for _, category := range categories {
		if config, exists := limits[category]; exists {
			fmt.Printf("%-12s: %3d requests per %v\n",
				category, config.MaxRequests, formatDuration(config.Window))
		}
	}
	fmt.Println("================================")
	fmt.Println()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

// PrintCurrentVisitors shows current visitor status (for debugging)
func PrintCurrentVisitors() {
	globalMu.RLock()
	defer globalMu.RUnlock()

	fmt.Printf("\n=== Current Visitors (%d) ===\n", len(visitors))
	for ip, v := range visitors {
		v.mu.RLock()
		fmt.Printf("IP: %s\n", ip)
		for category, count := range v.requests {
			lastSeen := v.lastSeen[category]
			fmt.Printf("  %s: %d requests (last seen: %s ago)\n",
				category, count, time.Since(lastSeen).Round(time.Second))
		}
		v.mu.RUnlock()
	}
	fmt.Println("==============================")
	fmt.Println()
}
