package models

import (
	"errors"
	"strings"
	"time"

	"forum/database"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Avatar       string    `json:"avatar"`
}

// Create user and insert it to the database
func (u *User) Create() error {
	// Validate input
	if strings.TrimSpace(u.Username) == "" || strings.TrimSpace(u.Email) == "" {
		return errors.New("username and email are required")
	}

	// Hash the password using bcrypt before storing
	if err := u.HashPassword(u.PasswordHash); err != nil {
		return err
	}

	// Check if user already exists
	exists, err := u.Exists()
	if err != nil {
		return err
	}
	if exists {
		return errors.New("user with this username or email already exists")
	}

	query := `
	INSERT INTO users (username, email, password_hash, avatar, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := database.GetDB().Exec(query, u.Username, u.Email, u.PasswordHash, u.Avatar, now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	u.ID = int(id)
	u.CreatedAt = now
	u.UpdatedAt = now
	return nil
}

// GetByUsername fills the user struct with data from the database taking username as input.
func (u *User) GetByUsername(username string) error {
	query := `SELECT id, username, email, password_hash, avatar, created_at, updated_at FROM users WHERE username = ?`
	row := database.GetDB().QueryRow(query, username)
	return row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Avatar, &u.CreatedAt, &u.UpdatedAt)
}

// GetByEmail fills the user struct with data from the database taking email as input.
func (u *User) GetByEmail(email string) error {
	query := `SELECT id, username, email, password_hash, avatar, created_at, updated_at FROM users WHERE email = ?`
	row := database.GetDB().QueryRow(query, email)
	return row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Avatar, &u.CreatedAt, &u.UpdatedAt)
}

// GetByID fills the user struct with data from the database taking id as input.
func (u *User) GetByID(id int) error {
	query := `SELECT id, username, email, password_hash, avatar, created_at, updated_at FROM users WHERE id = ?`
	row := database.GetDB().QueryRow(query, id)
	return row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Avatar, &u.CreatedAt, &u.UpdatedAt)
}

// Exists checks for duplicate users
func (u *User) Exists() (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE username = ? OR email = ?`
	var count int
	err := database.GetDB().QueryRow(query, u.Username, u.Email).Scan(&count)
	return count > 0, err
}

// HashPassword hashes a plain text password and stores it in PasswordHash
func (u *User) HashPassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return errors.New("password cannot be empty")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedBytes)
	return nil
}

// VerifyPassword checks if the provided password matches the stored hash
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// CheckPassword is an alias for VerifyPassword for consistency
func (u *User) CheckPassword(password string) bool {
	return u.VerifyPassword(password)
}

// UpdatePassword updates the user's password with a new hashed password
func (u *User) UpdatePassword(newPassword string) error {
	if strings.TrimSpace(newPassword) == "" {
		return errors.New("password cannot be empty")
	}

	// Hash the new password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
	now := time.Now()
	_, err = database.GetDB().Exec(query, string(hashedBytes), now, u.ID)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hashedBytes)
	u.UpdatedAt = now
	return nil
}

// UpdateUsername updates the user's username
func (u *User) UpdateUsername(newUsername string) error {
	if strings.TrimSpace(newUsername) == "" {
		return errors.New("username cannot be empty")
	}

	query := `UPDATE users SET username = ?, updated_at = ? WHERE id = ?`
	now := time.Now()
	_, err := database.GetDB().Exec(query, newUsername, now, u.ID)
	if err != nil {
		return err
	}

	u.Username = newUsername
	u.UpdatedAt = now
	return nil
}

// UpdateEmail updates the user's email
func (u *User) UpdateEmail(newEmail string) error {
	if strings.TrimSpace(newEmail) == "" {
		return errors.New("email cannot be empty")
	}

	query := `UPDATE users SET email = ?, updated_at = ? WHERE id = ?`
	now := time.Now()
	_, err := database.GetDB().Exec(query, newEmail, now, u.ID)
	if err != nil {
		return err
	}

	u.Email = newEmail
	u.UpdatedAt = now
	return nil
}

// GetAvatarURL returns the user's avatar URL or default if empty
func (u *User) GetAvatarURL() string {
	return strings.TrimSpace(u.Avatar) // empty string if no avatar
}

// UpdateAvatar updates the user's avatar
func (u *User) UpdateAvatar(avatarURL string) error {
	query := `UPDATE users SET avatar = ?, updated_at = ? WHERE id = ?`
	now := time.Now()

	_, err := database.GetDB().Exec(query, avatarURL, now, u.ID)
	if err != nil {
		return err
	}

	u.Avatar = avatarURL
	u.UpdatedAt = now
	return nil
}

// UpdateLastLogin update the last login timestamp for the user
func (u *User) UpdateLastLogin() error {
	query := `UPDATE users SET updated_at = ? WHERE id = ?`
	now := time.Now()

	_, err := database.GetDB().Exec(query, now, u.ID)
	if err != nil {
		return err
	}
	u.UpdatedAt = now
	return nil
}

// UpdateProfile updates multiple user fields at once
func (u *User) UpdateProfile(updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no fields to update")
	}

	var setParts []string
	var args []interface{}

	// Build dynamic query based on provided fields
	for field, value := range updates {
		switch field {
		case "username":
			if username, ok := value.(string); ok && strings.TrimSpace(username) != "" {
				setParts = append(setParts, "username = ?")
				args = append(args, username)
				u.Username = username
			}
		case "email":
			if email, ok := value.(string); ok && strings.TrimSpace(email) != "" {
				setParts = append(setParts, "email = ?")
				args = append(args, email)
				u.Email = email
			}
		case "avatar":
			if avatar, ok := value.(string); ok {
				setParts = append(setParts, "avatar = ?")
				args = append(args, avatar)
				u.Avatar = avatar
			}
		}
	}

	if len(setParts) == 0 {
		return errors.New("no valid fields to update")
	}

	// Add updated_at
	setParts = append(setParts, "updated_at = ?")
	now := time.Now()
	args = append(args, now)

	// Add WHERE clause
	args = append(args, u.ID)

	query := `UPDATE users SET ` + strings.Join(setParts, ", ") + ` WHERE id = ?`
	_, err := database.GetDB().Exec(query, args...)
	if err != nil {
		return err
	}

	u.UpdatedAt = now
	return nil
}

// GetPublicProfile returns user data safe for public viewing
func (u *User) GetPublicProfile() map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"username":   u.Username,
		"avatar":     u.GetAvatarURL(),
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}

// GetPrivateProfile returns user data including sensitive info (for profile owner)
func (u *User) GetPrivateProfile() map[string]interface{} {
	profile := u.GetPublicProfile()
	profile["email"] = u.Email
	return profile
}
