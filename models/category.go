package models

import (
	"errors"
	"strings"
	"time"

	"forum/database"
)

// Category represents forum category/section
type Category struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PostCount   int       `json:"post_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CategoryStats represents category statistics
type CategoryStats struct {
	TotalPosts     int        `json:"total_posts"`
	TotalComments  int        `json:"total_comments"`
	LastPostDate   *time.Time `json:"last_post_date"`
	LastPostTitle  string     `json:"last_post_title"`
	LastPostAuthor string     `json:"last_post_author"`
	ActiveUsers    int        `json:"active_users"`
}

// Create adds a new category to the database
func (c *Category) Create() error {
	// Validate category data
	if err := c.Validate(); err != nil {
		return err
	}

	// Check if category name already exists
	if exists, err := c.NameExists(); err != nil {
		return err
	} else if exists {
		return errors.New("category with this name already exists")
	}

	query := `
		INSERT INTO categories (name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`

	now := time.Now()
	result, err := database.GetDB().Exec(query, c.Name, c.Description, now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	c.ID = int(id)
	c.CreatedAt = now
	c.UpdatedAt = now

	return nil
}

// GetByID retrieves a category by its ID
func (c *Category) GetByID(id int) error {
	query := `
		SELECT c.id, c.name, c.description, c.created_at, c.updated_at,
		       COUNT(p.id) as post_count
		FROM categories c
		LEFT JOIN posts p ON c.id = p.category_id
		WHERE c.id = ?
		GROUP BY c.id, c.name, c.description, c.created_at, c.updated_at
	`

	row := database.GetDB().QueryRow(query, id)
	err := row.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt, &c.PostCount)
	return err
}

// GetByName retrieves a category by its name
func (c *Category) GetByName(name string) error {
	query := `
		SELECT c.id, c.name, c.description, c.created_at, c.updated_at,
		       COUNT(p.id) as post_count
		FROM categories c
		LEFT JOIN posts p ON c.id = p.category_id
		WHERE c.name = ?
		GROUP BY c.id, c.name, c.description, c.created_at, c.updated_at
	`

	row := database.GetDB().QueryRow(query, name)
	err := row.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt, &c.PostCount)
	return err
}

// GetAll retrieves all categories with post counts
func GetAllCategories() ([]Category, error) {
	var categories []Category

	query := `
			SELECT c.id, c.name, c.description, c.created_at, c.updated_at,
       		COUNT(DISTINCT pc.post_id) as post_count
			FROM categories c
			LEFT JOIN post_categories pc ON c.id = pc.category_id
			GROUP BY c.id, c.name, c.description, c.created_at, c.updated_at
			ORDER BY c.name

		`

	rows, err := database.GetDB().Query(query)
	if err != nil {
		return categories, err
	}
	defer rows.Close()

	for rows.Next() {
		var category Category
		err := rows.Scan(&category.ID, &category.Name, &category.Description,
			&category.CreatedAt, &category.UpdatedAt, &category.PostCount)
		if err != nil {
			continue
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// GetPopularCategories returns categories sorted by post count
func GetPopularCategories(limit int) ([]Category, error) {
	var categories []Category

	query := `
		SELECT c.id, c.name, c.description, c.created_at, c.updated_at,
		       COUNT(p.id) as post_count
		FROM categories c
		LEFT JOIN posts p ON c.id = p.category_id
		GROUP BY c.id, c.name, c.description, c.created_at, c.updated_at
		ORDER BY post_count DESC, c.name
		LIMIT ?
	`

	rows, err := database.GetDB().Query(query, limit)
	if err != nil {
		return categories, err
	}
	defer rows.Close()

	for rows.Next() {
		var category Category
		err := rows.Scan(&category.ID, &category.Name, &category.Description,
			&category.CreatedAt, &category.UpdatedAt, &category.PostCount)
		if err != nil {
			continue
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// Update modifies an existing category
func (c *Category) Update() error {
	// Validate category data
	if err := c.Validate(); err != nil {
		return err
	}

	// Check if new name conflicts with existing category (excluding current)
	var count int
	checkQuery := `SELECT COUNT(*) FROM categories WHERE name = ? AND id != ?`
	err := database.GetDB().QueryRow(checkQuery, c.Name, c.ID).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("category name already exists")
	}

	query := `
		UPDATE categories 
		SET name = ?, description = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err = database.GetDB().Exec(query, c.Name, c.Description, now, c.ID)
	if err != nil {
		return err
	}

	c.UpdatedAt = now
	return nil
}

// Delete removes a category from the database
func (c *Category) Delete() error {
	// Check if category has posts
	var postCount int
	countQuery := `SELECT COUNT(*) FROM posts WHERE category_id = ?`
	err := database.GetDB().QueryRow(countQuery, c.ID).Scan(&postCount)
	if err != nil {
		return err
	}

	if postCount > 0 {
		return errors.New("cannot delete category with existing posts")
	}

	query := `DELETE FROM categories WHERE id = ?`
	result, err := database.GetDB().Exec(query, c.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("category not found")
	}

	return nil
}

// GetStats returns detailed statistics for the category
func (c *Category) GetStats() (*CategoryStats, error) {
	stats := &CategoryStats{}

	// Get basic counts
	query := `
		SELECT 
			COUNT(DISTINCT p.id) as total_posts,
			COUNT(DISTINCT co.id) as total_comments
		FROM posts p
		LEFT JOIN comments co ON p.id = co.post_id
		WHERE p.category_id = ?
	`
	err := database.GetDB().QueryRow(query, c.ID).Scan(&stats.TotalPosts, &stats.TotalComments)
	if err != nil {
		return nil, err
	}

	// Get last post info
	lastPostQuery := `
		SELECT p.created_at, p.title, u.username
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.category_id = ?
		ORDER BY p.created_at DESC
		LIMIT 1
	`
	row := database.GetDB().QueryRow(lastPostQuery, c.ID)
	err = row.Scan(&stats.LastPostDate, &stats.LastPostTitle, &stats.LastPostAuthor)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, err
	}

	// Get active users count (posted in last 30 days)
	activeUsersQuery := `
		SELECT COUNT(DISTINCT p.user_id)
		FROM posts p
		WHERE p.category_id = ? 
		AND p.created_at >= datetime('now', '-30 days')
	`
	err = database.GetDB().QueryRow(activeUsersQuery, c.ID).Scan(&stats.ActiveUsers)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// NameExists checks if a category name already exists
func (c *Category) NameExists() (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM categories WHERE name = ?`
	err := database.GetDB().QueryRow(query, c.Name).Scan(&count)
	return count > 0, err
}

// Validate checks if category data is valid
func (c *Category) Validate() error {
	// Check name
	if len(strings.TrimSpace(c.Name)) == 0 {
		return errors.New("category name is required")
	}
	if len(c.Name) > 50 {
		return errors.New("category name cannot exceed 50 characters")
	}

	// Normalize name (lowercase, no spaces)
	c.Name = strings.ToLower(strings.TrimSpace(c.Name))
	c.Name = strings.ReplaceAll(c.Name, " ", "-")

	// Check description
	if len(c.Description) > 255 {
		return errors.New("category description cannot exceed 255 characters")
	}

	// Trim description
	c.Description = strings.TrimSpace(c.Description)

	return nil
}

// GetPostsInCategory returns paginated posts for this category
func (c *Category) GetPosts(limit, offset int, sortBy string) ([]Post, int, error) {
	filters := PostFilters{
		CategoryID: c.ID,
		SortBy:     sortBy,
		Limit:      limit,
		Offset:     offset,
	}

	return GetPosts(filters)
}

// UpdatePostCount recalculates and updates the cached post count
func (c *Category) UpdatePostCount() error {
	query := `
		UPDATE categories 
		SET updated_at = ?
		WHERE id = ?
	`
	now := time.Now()
	_, err := database.GetDB().Exec(query, now, c.ID)
	if err != nil {
		return err
	}

	// Refresh the post count in memory
	return c.refreshPostCount()
}

// refreshPostCount updates the in-memory post count
func (c *Category) refreshPostCount() error {
	query := `SELECT COUNT(*) FROM posts WHERE category_id = ?`
	return database.GetDB().QueryRow(query, c.ID).Scan(&c.PostCount)
}

// GetRecentActivity returns recent posts and comments in this category
func (c *Category) GetRecentActivity(limit int) ([]map[string]interface{}, error) {
	var activity []map[string]interface{}

	// Get recent posts
	postQuery := `
		SELECT 'post' as type, p.id, p.title as content, u.username, p.created_at
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.category_id = ?
		ORDER BY p.created_at DESC
		LIMIT ?
	`

	rows, err := database.GetDB().Query(postQuery, c.ID, limit)
	if err != nil {
		return activity, err
	}
	defer rows.Close()

	for rows.Next() {
		var actType, content, username string
		var id int
		var createdAt time.Time

		err := rows.Scan(&actType, &id, &content, &username, &createdAt)
		if err != nil {
			continue
		}

		activity = append(activity, map[string]interface{}{
			"type":       actType,
			"id":         id,
			"content":    content,
			"username":   username,
			"created_at": createdAt,
		})
	}

	return activity, nil
}

// IsEmpty checks if category has any posts
func (c *Category) IsEmpty() (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM posts WHERE category_id = ?`
	err := database.GetDB().QueryRow(query, c.ID).Scan(&count)
	return count == 0, err
}

// CanDelete checks if category can be safely deleted
func (c *Category) CanDelete() (bool, string, error) {
	isEmpty, err := c.IsEmpty()
	if err != nil {
		return false, "", err
	}

	if !isEmpty {
		return false, "Category contains posts and cannot be deleted", nil
	}

	return true, "", nil
}