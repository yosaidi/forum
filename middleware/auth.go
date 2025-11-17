package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
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
		username, hasUser := GetUsernameFromContext(r)
		userInfo := "anonymous"
		if hasUser {
			userInfo = username
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
// cause all default to 200 which is delulu
// WriteHeader captures the status code when it's written
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}


// Recovery middleware catches panics and returns 500
func Recovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic with stack trace (server-side only)
				log.Printf("PANIC: %v\n%s", err, debug.Stack())
				
				// Return clean 500 to client (no stack trace leak)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "Internal server error",
					"data":    nil,
				})
			}
		}()
		next(w, r)
	}
}