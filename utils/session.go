package utils

import (
	"database/sql"
	"net/http"
	"time"

	"forum/config"
	"forum/database"

	"github.com/google/uuid"
)

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionDuration defines how long sessions last (24 hours)
const SessionDuration = 24 * time.Hour

// CookieName is the name of the session cookie
const CookieName = "forum_session"

// CreateSession creates a new session for a user
func CreateSession(userID int) (*Session, error) {
	// Delete any existing session for the user (single-session login)
	if err := DeleteUserSessions(userID); err != nil {
		return nil, err
	}

	session := &Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		ExpiresAt: time.Now().Add(SessionDuration),
		CreatedAt: time.Now(),
	}

	// Store session in database
	query := `
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`

	_, err := database.GetDB().Exec(query, session.ID, session.UserID, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves a session by its ID
func GetSession(sessionID string) (*Session, error) {
	session := &Session{}

	query := `
		SELECT id, user_id, expires_at, created_at 
		FROM sessions 
		WHERE id = ? AND expires_at > ?
	`

	err := database.GetDB().QueryRow(query, sessionID, time.Now()).Scan(
		&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteSession removes a session from the database
func DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := database.GetDB().Exec(query, sessionID)
	return err
}

// DeleteUserSessions removes all sessions for a specific user
func DeleteUserSessions(userID int) error {
	query := `DELETE FROM sessions WHERE user_id = ?`
	_, err := database.GetDB().Exec(query, userID)
	return err
}

// RefreshSession extends the expiration time of a session
func RefreshSession(sessionID string) (*Session, error) {
	newExpiration := time.Now().Add(SessionDuration)

	query := `
		UPDATE sessions 
		SET expires_at = ? 
		WHERE id = ? AND expires_at > ?
	`

	result, err := database.GetDB().Exec(query, newExpiration, sessionID, time.Now())
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rowsAffected == 0 {
		return nil, sql.ErrNoRows
	}

	// Return updated session
	return GetSession(sessionID)
}

// SetSessionCookie sets the session cookie in the HTTP response
func SetSessionCookie(w http.ResponseWriter, session *Session) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    session.ID,
		Expires:  session.ExpiresAt,
		HttpOnly: true,                    // Prevent XSS attacks
		Secure:   !config.IsDevelopment(), // HTTPS only in production
		SameSite: http.SameSiteLaxMode,    // CSRF protection
		Path:     "/",                     // Available site-wide
	}

	http.SetCookie(w, cookie)
}

// ClearSessionCookie removes the session cookie
func ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // Expire in the past
		HttpOnly: true,
		Secure:   !config.IsDevelopment(),
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	}

	http.SetCookie(w, cookie)
}

// GetSessionFromRequest extracts session from HTTP request
func GetSessionFromRequest(r *http.Request) (*Session, error) {
	// Get session cookie
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return nil, err
	}

	// Get session from database
	session, err := GetSession(cookie.Value)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetUserFromSession gets user ID from session in request
func GetUserFromSession(r *http.Request) (int, error) {
	session, err := GetSessionFromRequest(r)
	if err != nil {
		return 0, err
	}

	return session.UserID, nil
}

// GetUserIDFromSession is an alias for GetUserFromSession for consistency
func GetUserIDFromSession(r *http.Request) (int, error) {
	return GetUserFromSession(r)
}

// IsLoggedIn checks if request has valid session
func IsLoggedIn(r *http.Request) bool {
	_, err := GetSessionFromRequest(r)
	return err == nil
}

// RequireLogin middleware function to check authentication
func RequireLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := GetSessionFromRequest(r)
		if err != nil {
			Unauthorized(w, "Please log in to access this resource")
			return
		}

		// Optional: Refresh session on activity
		if time.Until(session.ExpiresAt) < SessionDuration/2 {
			refreshedSession, err := RefreshSession(session.ID)
			if err == nil {
				SetSessionCookie(w, refreshedSession)
			}
		}

		// Continue to next handler
		next(w, r)
	}
}

// GetCurrentUser gets current user info from session
func GetCurrentUser(r *http.Request) (int, string, error) {
	session, err := GetSessionFromRequest(r)
	if err != nil {
		return 0, "", err
	}

	// Get username from database
	query := `SELECT username FROM users WHERE id = ?`
	var username string
	err = database.GetDB().QueryRow(query, session.UserID).Scan(&username)
	if err != nil {
		return 0, "", err
	}

	return session.UserID, username, nil
}

// CleanupExpiredSessions removes expired sessions from database
func CleanupExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at <= ?`
	_, err := database.GetDB().Exec(query, time.Now())
	return err
}

// SessionInfo provides session statistics
type SessionInfo struct {
	TotalSessions   int `json:"total_sessions"`
	ActiveSessions  int `json:"active_sessions"`
	ExpiredSessions int `json:"expired_sessions"`
}

// GetSessionStats returns session statistics (admin function)
func GetSessionStats() (*SessionInfo, error) {
	info := &SessionInfo{}

	// Total sessions
	err := database.GetDB().QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&info.TotalSessions)
	if err != nil {
		return nil, err
	}

	// Active sessions
	err = database.GetDB().QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE expires_at > ?`,
		time.Now(),
	).Scan(&info.ActiveSessions)
	if err != nil {
		return nil, err
	}

	// Expired sessions
	info.ExpiredSessions = info.TotalSessions - info.ActiveSessions

	return info, nil
}

// ValidateSessionID checks if session ID format is valid
func ValidateSessionID(sessionID string) bool {
	// Check if it's a valid UUID format
	_, err := uuid.Parse(sessionID)
	return err == nil
}
