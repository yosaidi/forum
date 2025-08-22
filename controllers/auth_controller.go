package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"forum/models"
	"forum/utils"
)

// RegisterRequest represents the JSON structure for user registration
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents the JSON structure for user login
type LoginRequest struct {
	Username string `json:"username"` // Can be username or email
	Password string `json:"password"`
}

// AuthResponse respresents the response after successful authentification
type AuthResponse struct {
	User    UserResponse `json:"user"`
	Session string       `json:"session"`
}

// UserResponse respresents user data sent to client (no sensitive info)
type UserResponse struct {
	ID       int       `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Avatar   string    `json:"avatar"`
	JoinedAt time.Time `json:"joined_at"`
}

// RegisterController handles user registration
func RegisterController(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		utils.MethodNotAllowed(w, "Only POST method allowed")
		return
	}

	// Parse JSON request body
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	// Validate registration form
	if errors := utils.ValidateRegistrationForm(req.Username, req.Email, req.Password); errors.HasErrors() {
		utils.ValidationError(w, errors)
		return
	}

	// Check if user already exists
	existingUser := models.User{}
	err := existingUser.GetByUsername(req.Username)
	if err == nil {
		utils.BadRequest(w, "Username already taken")
		return
	} else if err != sql.ErrNoRows {
		utils.InternalServerError(w, "Database error")
		return
	}

	// Create new user
	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: req.Password, // will be hashed in user.Create()
	}

	if err := user.Create(); err != nil {
		utils.InternalServerError(w, "Failed to created user account")
		return
	}

	// Create session for new user
	session, err := utils.CreateSession(user.ID)
	if err != nil {
		utils.InternalServerError(w, "Account created but login failed")
		return
	}

	// Set session cookie
	utils.SetSessionCookie(w, session)

	// Prepare response
	UserResponse := UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Avatar:   user.Avatar,
		JoinedAt: user.CreatedAt,
	}

	authResponse := AuthResponse{
		User:    UserResponse,
		Session: session.ID,
	}

	utils.Created(w, "Account created successfully", authResponse)
}

// LoginController handles user login
func LoginController(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		utils.MethodNotAllowed(w, "Only POST method allowed")
		return
	}

	// Parse JSON request body
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	if errors := utils.ValidateLoginForm(req.Username, req.Password); errors.HasErrors() {
		utils.ValidationError(w, errors)
		return
	}

	// Find user by username or email
	user := models.User{}
	var err error

	// Try to find by username first
	if err = user.GetByUsername(req.Username); err != nil {
		// If not found , try by email
		if err = user.GetByEmail(req.Username); err != nil {
			utils.Unauthorized(w, "Invalid username/email or password")
			return
		}
	}

	// Verify password
	if !user.CheckPassword(req.Password) {
		utils.Unauthorized(w, "Invalid username/email or password")
		return
	}

	// Update last login time
	user.UpdateLastLogin()

	// Create session
	session, err := utils.CreateSession(user.ID)
	if err != nil {
		utils.InternalServerError(w, "Login failed")
		return
	}

	// Set session cookie
	utils.SetSessionCookie(w, session)

	// Prepare response
	userResponse := UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Avatar:   user.Avatar,
		JoinedAt: user.CreatedAt,
	}

	authResponse := AuthResponse{
		User:    userResponse,
		Session: session.ID,
	}

	utils.Success(w, "Login successful", authResponse)
}

// LogoutController handles user logout
func LogoutController(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		utils.MethodNotAllowed(w, "Only POST method allowed")
		return
	}

	sessionID, err := r.Cookie("forum_session")
	if err != nil {
		utils.Success(w, "Logged out successfully", nil) // Already logged out
		return
	}

	// Delete session from database
	if err := utils.DeleteSession(sessionID.Value); err != nil {
		// Log error but don't fail the logout
		// utils.LogError("Failed to delete session", err)
	}

	// Clear session cookie
	utils.ClearSessionCookie(w)

	utils.Success(w, "Logged out successfully", nil)
}

// MeController returns current user info
func MeController(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	// Get user from session
	userID, err := utils.GetUserIDFromSession(r)
	if err != nil {
		utils.Unauthorized(w, "Not authenticated")
		return
	}

	// Get user details
	user := models.User{}
	if err := user.GetByID(userID); err != nil {
		utils.NotFound(w, "User not found")
		return
	}

	// Prepare response
	userResponse := UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Avatar:   user.Avatar,
		JoinedAt: user.CreatedAt,
	}

	utils.Success(w, "User data retrieved", userResponse)
}

// RefreshSessionController extends current session
func RefreshSessionController(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		utils.MethodNotAllowed(w, "Only POST method allowed")
		return
	}

	// Get current session
	sessionCookie, err := r.Cookie("forum_session")
	if err != nil {
		utils.Unauthorized(w, "No active session")
		return
	}

	// Refresh session
	newSession, err := utils.RefreshSession(sessionCookie.Value)
	if err != nil {
		utils.Unauthorized(w, "Session expired")
		return
	}

	// Update cookie with new session
	utils.SetSessionCookie(w, newSession)

	utils.Success(w, "Session refreshed", map[string]string{
		"session": newSession.ID,
	})
}

// CheckUsernameController checks if username is available
func CheckUsernameController(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	// Get username from query parameters
	username := r.URL.Query().Get("username")
	if username == "" {
		utils.BadRequest(w, "Username parameter is required")
		return
	}

	// Validate username format
	if err := utils.ValidateUsername(username); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Check if username exists
	user := models.User{}
	isAvailable := user.GetByUsername(username) != nil // nil means not found = available

	utils.Success(w, "Username availability checked", map[string]bool{
		"available": isAvailable,
	})
}

// CheckEmailController checks if email is available
func CheckEmailController(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	// Get email from query parameters
	email := r.URL.Query().Get("email")
	if email == "" {
		utils.BadRequest(w, "Email parameter is required")
		return
	}

	// Validate email format
	if err := utils.ValidateEmail(email); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Check if email exists
	user := models.User{}
	isAvailable := user.GetByEmail(email) != nil // nil means not found = available

	utils.Success(w, "Email availability checked", map[string]bool{
		"available": isAvailable,
	})
}
