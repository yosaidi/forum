package utils

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

// ValidationErrors holds field-specific validation errors
type ValidationErrors map[string]string

// Add adds a validation error for a specific field
func (ve ValidationErrors) Add(field, message string) {
	ve[field] = message
}

// HasErrors checks if there are any validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// ToError converts validation errors to a single error
func (ve ValidationErrors) ToError() error {
	if !ve.HasErrors() {
		return nil
	}

	var messages []string
	for field, message := range ve {
		messages = append(messages, field+": "+message)
	}

	return errors.New(strings.Join(messages, "; "))
}

// Email validation regex pattern
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Username validation regex pattern (alphanumeric, underscore, hyphen)
var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateEmail checks if an email address is valid
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if len(email) > 254 {
		return errors.New("email is too long (max 254 characters)")
	}

	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}

	return nil
}

// ValidateEmail returns string error message (for individual validation)
func ValidateEmailString(email string) string {
	if err := ValidateEmail(email); err != nil {
		return err.Error()
	}
	return ""
}

// ValidateUsername checks if a username is valid
func ValidateUsername(username string) error {
	if username == "" {
		return errors.New("username is required")
	}

	if len(username) < 3 {
		return errors.New("username must be at least 3 characters long")
	}

	if len(username) > 50 {
		return errors.New("username is too long (max 50 characters)")
	}

	if !usernameRegex.MatchString(username) {
		return errors.New("username can only contain letters, numbers, underscores, and hyphens")
	}

	// Check if username starts or ends with special characters
	if strings.HasPrefix(username, "_") || strings.HasPrefix(username, "-") ||
		strings.HasSuffix(username, "_") || strings.HasSuffix(username, "-") {
		return errors.New("username cannot start or end with underscores or hyphens")
	}

	return nil
}

// ValidateUsername returns string error message (for individual validation)
func ValidateUsernameString(username string) string {
	if err := ValidateUsername(username); err != nil {
		return err.Error()
	}
	return ""
}

// ValidatePassword checks if a password meets security requirements
func ValidatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}

	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return errors.New("password is too long (max 128 characters)")
	}

	// Check for at least one uppercase letter
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}

	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}

	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}

	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

// ValidatePostTitle checks if a post title is valid
func ValidatePostTitle(title string) error {
	title = strings.TrimSpace(title)

	if title == "" {
		return errors.New("post title is required")
	}

	if len(title) < 5 {
		return errors.New("post title must be at least 5 characters long")
	}

	if len(title) > 255 {
		return errors.New("post title is too long (max 255 characters)")
	}

	return nil
}

// ValidatePostContent checks if post content is valid
func ValidatePostContent(content string) error {
	content = strings.TrimSpace(content)

	if content == "" {
		return errors.New("post content is required")
	}

	if len(content) < 10 {
		return errors.New("post content must be at least 10 characters long")
	}

	if len(content) > 10000 {
		return errors.New("post content is too long (max 10,000 characters)")
	}

	return nil
}

// ValidateCommentContent checks if comment content is valid
func ValidateCommentContent(content string) error {
	content = strings.TrimSpace(content)

	if content == "" {
		return errors.New("comment content is required")
	}

	if len(content) < 1 {
		return errors.New("comment cannot be empty")
	}

	if len(content) > 1000 {
		return errors.New("comment is too long (max 1,000 characters)")
	}

	return nil
}

// ValidateRegistrationForm validates user registration data
func ValidateRegistrationForm(username, email, password string) ValidationErrors {
	errors := make(ValidationErrors)

	// Validate username
	if err := ValidateUsername(username); err != nil {
		errors.Add("username", err.Error())
	}

	// Validate email
	if err := ValidateEmail(email); err != nil {
		errors.Add("email", err.Error())
	}

	// Validate password
	if err := ValidatePassword(password); err != nil {
		errors.Add("password", err.Error())
	}

	return errors
}

// ValidateLoginForm validates user login data
func ValidateLoginForm(username, password string) ValidationErrors {
	errors := make(ValidationErrors)

	// Validate username (can be email or username)
	if username == "" {
		errors.Add("username", "username or email is required")
	}

	// Validate password
	if password == "" {
		errors.Add("password", "password is required")
	}

	return errors
}

// ValidatePostForm validates post creation/update data
func ValidatePostForm(title, content string, categoryID int) ValidationErrors {
	errors := make(ValidationErrors)

	// Validate title
	if err := ValidatePostTitle(title); err != nil {
		errors.Add("title", err.Error())
	}

	// Validate content
	if err := ValidatePostContent(content); err != nil {
		errors.Add("content", err.Error())
	}

	// Validate category
	if categoryID <= 0 {
		errors.Add("category", "please select a valid category")
	}

	return errors
}

// ValidateCommentForm validates comment creation/update data
func ValidateCommentForm(content string) ValidationErrors {
	errors := make(ValidationErrors)

	// Validate content
	if err := ValidateCommentContent(content); err != nil {
		errors.Add("content", err.Error())
	}

	return errors
}

// SanitizeString removes dangerous characters and trims whitespace
func SanitizeString(input string) string {
	// Remove null bytes and other control characters
	cleaned := strings.Map(func(r rune) rune {
		if r == 0 || (r < 32 && r != '\n' && r != '\r' && r != '\t') {
			return -1
		}
		return r
	}, input)

	// Trim whitespace
	return strings.TrimSpace(cleaned)
}

// IsValidVoteType checks if vote type is valid
func IsValidVoteType(voteType string) bool {
	return voteType == "like" || voteType == "dislike"
}

// ValidatePagination validates pagination parameters
func ValidatePagination(page, limit int) (int, int, error) {
	// Default values
	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 20
	}

	// Maximum limits
	if limit > 100 {
		return 0, 0, errors.New("limit cannot exceed 100 items per page")
	}

	// Calculate offset
	offset := (page - 1) * limit

	return limit, offset, nil
}

// ValidateID checks if an ID is valid (positive integer)
func ValidateID(id int, fieldName string) error {
	if id <= 0 {
		return errors.New(fieldName + " must be a positive integer")
	}
	return nil
}
