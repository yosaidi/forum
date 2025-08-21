package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"forum/utils"
)

// ContextKey type for context keys to avoid collisions
type ContextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// UsernameKey is the context key for username
	UsernameKey ContextKey = "username"
	// SessionKey is the context key for session
	SessionKey ContextKey = "session"
)

// RequireAuth middleware requires user authentication
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session from request
		session, err := utils.GetSessionFromRequest(r)
		if err != nil {
			utils.Unauthorized(w, "Authentication required. Please log in.")
			return
		}

		// Check if session is still valid (extra safety check)
		if session.ExpiresAt.Before(time.Now()) {
			// Session expired, clear cookie
			utils.ClearSessionCookie(w)
			utils.Unauthorized(w, "Session expired. Please log in again.")
			return
		}

		// Get user information
		userID, username, err := utils.GetCurrentUser(r)
		if err != nil {
			utils.Unauthorized(w, "Invalid session. Please log in again.")
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, UsernameKey, username)
		ctx = context.WithValue(ctx, SessionKey, session)

		// Optional: Refresh session if it's halfway to expiration
		if time.Until(session.ExpiresAt) < utils.SessionDuration/2 {
			refreshedSession, err := utils.RefreshSession(session.ID)
			if err == nil {
				utils.SetSessionCookie(w, refreshedSession)
				ctx = context.WithValue(ctx, SessionKey, refreshedSession)
			}
		}

		// Continue with authenticated context
		next(w, r.WithContext(ctx))
	}
}

// OptionalAuth middleware provides user info if logged in, but doesn't require it
func OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to get session from request
		session, err := utils.GetSessionFromRequest(r)
		if err != nil {
			// No session, continue without authentication
			next(w, r)
			return
		}

		// Check if session is still valid
		if session.ExpiresAt.Before(time.Now()) {
			// Session expired, clear cookie and continue without auth
			utils.ClearSessionCookie(w)
			next(w, r)
			return
		}

		// Get user information
		userID, username, err := utils.GetCurrentUser(r)
		if err != nil {
			// Invalid session, continue without auth
			next(w, r)
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, UsernameKey, username)
		ctx = context.WithValue(ctx, SessionKey, session)

		// Continue with authenticated context
		next(w, r.WithContext(ctx))
	}
}

// RequireOwnership middleware checks if user owns a resource
// This is a higher-order function that returns middleware
func RequireOwnership(getResourceUserID func(r *http.Request) (int, error)) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Get current user ID from context (must be called after RequireAuth)
			currentUserID, ok := r.Context().Value(UserIDKey).(int)
			if !ok {
				utils.InternalServerError(w, "Authentication context not found")
				return
			}

			// Get the resource owner ID
			resourceUserID, err := getResourceUserID(r)
			if err != nil {
				utils.NotFound(w, "Resource not found")
				return
			}

			// Check ownership
			if currentUserID != resourceUserID {
				utils.Forbidden(w, "You don't have permission to access this resource")
				return
			}

			// User owns the resource, continue
			next(w, r)
		}
	}
}

// GetUserIDFromContext retrieves user ID from request context
func GetUserIDFromContext(r *http.Request) (int, bool) {
	userID, ok := r.Context().Value(UserIDKey).(int)
	return userID, ok
}

// GetUsernameFromContext retrieves username from request context
func GetUsernameFromContext(r *http.Request) (string, bool) {
	username, ok := r.Context().Value(UsernameKey).(string)
	return username, ok
}

// GetSessionFromContext retrieves session from request context
func GetSessionFromContext(r *http.Request) (*utils.Session, bool) {
	session, ok := r.Context().Value(SessionKey).(*utils.Session)
	return session, ok
}

// RequireValidMethod middleware ensures only specific HTTP methods are allowed
func RequireValidMethod(allowedMethods ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Check if method is allowed
			for _, method := range allowedMethods {
				if r.Method == method {
					next(w, r)
					return
				}
			}

			// Method not allowed
			w.Header().Set("Allow", joinMethods(allowedMethods))
			utils.Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

// RateLimit middleware (basic implementation)
// In production, use Redis or more sophisticated rate limiting
func RateLimit(maxRequests int, timeWindow time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	// Simple in-memory rate limiter (not suitable for production with multiple servers)
	clients := make(map[string][]time.Time)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			clientIP := getClientIP(r)

			// Clean old requests
			now := time.Now()
			if requests, exists := clients[clientIP]; exists {
				var validRequests []time.Time
				for _, reqTime := range requests {
					if now.Sub(reqTime) < timeWindow {
						validRequests = append(validRequests, reqTime)
					}
				}
				clients[clientIP] = validRequests
			}

			// Check rate limit
			if len(clients[clientIP]) >= maxRequests {
				utils.Error(w, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.")
				return
			}

			// Add current request
			clients[clientIP] = append(clients[clientIP], now)

			// Continue
			next(w, r)
		}
	}
}

// ContentTypeJSON middleware ensures request has JSON content type
func ContentTypeJSON(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip for GET requests
		if r.Method == http.MethodGet {
			next(w, r)
			return
		}

		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			utils.BadRequest(w, "Content-Type must be application/json")
			return
		}

		next(w, r)
	}
}

// LogRequests middleware logs HTTP requests (basic logging)
func LogRequests(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a wrapped response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next(wrapped, r)

		// Log request
		duration := time.Since(start)

		// Get user info if available
		userID, hasUser := GetUserIDFromContext(r)
		userInfo := "anonymous"
		if hasUser {
			userInfo = fmt.Sprintf("user:%d", userID)
		}

		log.Printf("%s %s %d %v %s %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			getClientIP(r),
			userInfo,
		)
	}
}

// Helper functions

// joinMethods joins HTTP methods for Allow header
func joinMethods(methods []string) string {
	return strings.Join(methods, ", ")
}

// getClientIP gets client IP address from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Use RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}

	return ip
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
