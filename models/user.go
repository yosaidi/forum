package models

import (
	"time"

	"forum/database"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Create user and insert it to the database
func (u *User) Create() error {
	query := `
	INSERT INTO users (username, email, password_hash, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := database.GetDB().Exec(query, u.Username, u.Email, u.PasswordHash, now, now)
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
	query := `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE username = ?`
	row := database.GetDB().QueryRow(query, username)
	return row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
}

// GetByUsername fills the user struct with data from the database taking email as input.
func (u *User) GetByEmail(email string) error {
	query := `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE email = ?`
	row := database.GetDB().QueryRow(query, email)
	return row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
}

// GetByUsername fills the user struct with data from the database taking id as input.
func (u *User) GetByID(id int) error {
	query := `SELECT id, username, email, password_hash, created_at, updated_at FROM users WHERE id = ?`
	row := database.GetDB().QueryRow(query, id)
	return row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
}

// Exists checks for duplicate users
func (u *User) Exists() (bool, error) {
	query := `SELECT COUNT(*) FROM users WHERE username = ? OR email = ?`
	var count int
	err := database.GetDB().QueryRow(query, u.Username, u.Email).Scan(&count)
	return count > 0, err
}
