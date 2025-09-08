package middleware

import (
	"fmt"
	"log"
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
		"auth":       {MaxRequests: 5, Window: 15 * time.Minute}, // Very restrictive for testing
		"posts":      {MaxRequests: 10, Window: time.Minute},
		"comments":   {MaxRequests: 15, Window: time.Minute},
		"users":      {MaxRequests: 20, Window: time.Minute},
		"categories": {MaxRequests: 50, Window: time.Minute},
		"default":    {MaxRequests: 10, Window: time.Minute},
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

// cleanup old visitors every 1 minute (more frequent for testing)
func cleanupVisitors() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		globalMu.Lock()
		now := time.Now()
		toDelete := make([]string, 0)

		for ip, visitor := range visitors {
			visitor.mu.RLock()
			shouldDelete := true

			for category, lastSeen := range visitor.lastSeen {
				config := getRateLimitConfig(category)
				if now.Sub(lastSeen) <= config.Window {
					shouldDelete = false
					break
				}
			}

			visitor.mu.RUnlock()

			if shouldDelete {
				toDelete = append(toDelete, ip)
			} else {
				visitor.mu.Lock()
				for category, lastSeen := range visitor.lastSeen {
					config := getRateLimitConfig(category)
					if now.Sub(lastSeen) > config.Window {
						delete(visitor.requests, category)
						delete(visitor.lastSeen, category)
					}
				}
				visitor.mu.Unlock()
			}
		}

		for _, ip := range toDelete {
			delete(visitors, ip)
		}
		globalMu.Unlock()
	}
}

// extractIP extracts the IP address from RemoteAddr
func extractIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

// getVisitor retrieves the visitor for a given IP address
func getVisitor(ip string) *visitor {
	globalMu.Lock()
	defer globalMu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		if len(visitors) >= maxVisitors {
			var oldestIP string
			var oldestTime time.Time = time.Now()

			for visitorIP, visitorData := range visitors {
				visitorData.mu.RLock()
				for _, lastSeen := range visitorData.lastSeen {
					if lastSeen.Before(oldestTime) {
						oldestTime = lastSeen
						oldestIP = visitorIP
					}
				}
				visitorData.mu.RUnlock()
			}

			if oldestIP != "" {
				delete(visitors, oldestIP)
			}
		}

		v = &visitor{
			requests: make(map[string]int),
			lastSeen: make(map[string]time.Time),
		}
		visitors[ip] = v
	}
	return v
}

// getRateLimitConfig gets the rate limit configuration for a category
func getRateLimitConfig(category string) RateLimitConfig {
	if config, exists := rateLimits[category]; exists {
		return config
	}
	return rateLimits["default"]
}

// determineCategory extracts the endpoint category from the request path
func determineCategory(path string) string {
	if path == "" || path == "/" {
		return "default"
	}

	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	// Extract the first segment as category
	if idx := strings.Index(path, "/"); idx != -1 {
		category := path[:idx]
		return category
	}

	return path
}

// RateLimitForCategory creates a rate limiter for a specific category
func RateLimitForCategory(category string) func(http.Handler) http.Handler {
	startCleanup()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r.RemoteAddr)
			visitor := getVisitor(ip)
			config := getRateLimitConfig(category)

			visitor.mu.Lock()
			defer visitor.mu.Unlock()

			now := time.Now()

			// Check if this category's window has expired
			if lastSeen, exists := visitor.lastSeen[category]; exists {
				if now.Sub(lastSeen) > config.Window {
					visitor.requests[category] = 0
				}
			}

			// Increment request count for this category
			visitor.requests[category]++
			visitor.lastSeen[category] = now

			currentCount := visitor.requests[category]

			// Rate-limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.MaxRequests))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(config.MaxRequests-currentCount))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(visitor.lastSeen[category].Add(config.Window).Unix(), 10))

			// Check rate limit
			if currentCount > config.MaxRequests {
				message := fmt.Sprintf("Rate limit exceeded for %s. Max %d requests per %v. Please slow down.",
					category, config.MaxRequests, config.Window)
				utils.TooManyRequests(w, message)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit is the general rate limiter that automatically determines category
func RateLimit(next http.Handler) http.Handler {
	startCleanup()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		category := "default"

		// Handle API routes
		if strings.HasPrefix(path, "/api/") {
			apiPath := strings.TrimPrefix(path, "/api/")
			category = determineCategory(apiPath)
		} else {
			category = determineCategory(path)
		}

		// Apply rate limiting for the determined category
		limiter := RateLimitForCategory(category)
		limiter(next).ServeHTTP(w, r)
	})
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

// GetRateLimits returns current rate limit configurations
func GetRateLimits() map[string]RateLimitConfig {
	globalMu.RLock()
	defer globalMu.RUnlock()

	result := make(map[string]RateLimitConfig)
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
	for _, visitor := range visitors {
		visitor.mu.RLock()
		for category := range visitor.requests {
			categoryStats[category]++
		}
		visitor.mu.RUnlock()
	}
	stats["categories"] = categoryStats

	return stats
}

// SetupCustomRateLimits - Production-ready rate limits
func SetupCustomRateLimits() {
	// Authentication endpoints - prevent brute force attacks
	SetRateLimit("auth", 5, 5*time.Minute) // 5 auth attempts per 5 minutes

	// Posts - prevent spam while allowing normal posting
	SetRateLimit("posts", 10, time.Hour) // 10 posts per hour

	// Comments - encourage discussion but prevent spam
	SetRateLimit("comments", 30, time.Hour) // 30 comments per hour (1 every 2 minutes)

	// User operations - profile views, updates, avatar uploads
	SetRateLimit("users", 60, time.Minute) // 60 user operations per minute

	// Categories - mostly read operations, be generous
	SetRateLimit("categories", 100, time.Minute) // 100 category requests per minute

	// Default fallback for any other endpoints
	SetRateLimit("default", 20, time.Minute) // 20 requests per minute

	log.Println("Applied production rate limits successfully")
}

// PrintRateLimitConfig displays current rate limit configuration
func PrintRateLimitConfig() {
	limits := GetRateLimits()
	fmt.Println()
	fmt.Println("=== Rate Limit Configuration ===")

	for category, config := range limits {
		fmt.Printf("%-12s: %3d requests per %v\n",
			category, config.MaxRequests, formatDuration(config.Window))
	}
	fmt.Println("=================================")
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

// SetMaxVisitors allows configuration of maximum visitors
func SetMaxVisitors(max int) {
	globalMu.Lock()
	defer globalMu.Unlock()
	maxVisitors = max
}

// PrintCurrentVisitors shows current visitor status (for debugging)
func PrintCurrentVisitors() {
	globalMu.RLock()
	defer globalMu.RUnlock()

	fmt.Printf("\n=== Current Visitors (%d) ===\n", len(visitors))
	for ip, visitor := range visitors {
		visitor.mu.RLock()
		fmt.Printf("IP: %s\n", ip)
		for category, count := range visitor.requests {
			lastSeen := visitor.lastSeen[category]
			fmt.Printf("  %s: %d requests (last seen: %s ago)\n",
				category, count, time.Since(lastSeen).Round(time.Second))
		}
		visitor.mu.RUnlock()
	}
	fmt.Println("===============================")
}
