package models

import (
	"errors"
	"strings"
	"time"

	"forum/database"
)

// Comment represents a comment on a post
type Comment struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	PostID    int       `json:"post_id"`
	Likes     int       `json:"likes"`
	Dislikes  int       `json:"dislikes"`
	UserVote  *string   `json:"user_vote"` // "like", "dislike", or nil
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Create adds a new comment to the database
func (c *Comment) Create() error {
	// Validate comment content
	if err := c.Validate(); err != nil {
		return err
	}

	query := `
		INSERT INTO comments (content, user_id, post_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := database.GetDB().Exec(query, c.Content, c.UserID, c.PostID, now, now)
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

// GetByID retrieves a comment by its ID with optional user vote info
func (c *Comment) GetByID(id int, userID *int) error {
	query := `
		SELECT c.id, c.content, c.user_id, u.username, c.post_id, 
		       c.likes, c.dislikes, c.created_at, c.updated_at
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.id = ?
	`

	row := database.GetDB().QueryRow(query, id)
	err := row.Scan(&c.ID, &c.Content, &c.UserID, &c.Username, &c.PostID,
		&c.Likes, &c.Dislikes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return err
	}

	// Get user vote if logged in
	if userID != nil {
		c.getUserVote(*userID)
	}

	return nil
}

// GetCommentsByPostID retrieves all comments for a specific post
func GetCommentsByPostID(postID int, userID *int, limit, offset int) ([]Comment, error) {
	var comments []Comment

	query := `
		SELECT c.id, c.content, c.user_id, u.username, c.post_id,
		       c.likes, c.dislikes, c.created_at, c.updated_at
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC
		LIMIT ? OFFSET ?
	`

	rows, err := database.GetDB().Query(query, postID, limit, offset)
	if err != nil {
		return comments, err
	}
	defer rows.Close()

	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.Content, &comment.UserID, &comment.Username,
			&comment.PostID, &comment.Likes, &comment.Dislikes,
			&comment.CreatedAt, &comment.UpdatedAt)
		if err != nil {
			continue
		}

		// Get user vote if logged in
		if userID != nil {
			comment.getUserVote(*userID)
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

// GetUserComments retrieves all comments made by a specific user
func GetUserComments(userID int, limit, offset int) ([]Comment, error) {
	var comments []Comment

	query := `
		SELECT c.id, c.content, c.user_id, u.username, c.post_id,
		       c.likes, c.dislikes, c.created_at, c.updated_at
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.user_id = ?
		ORDER BY c.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := database.GetDB().Query(query, userID, limit, offset)
	if err != nil {
		return comments, err
	}
	defer rows.Close()

	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.Content, &comment.UserID, &comment.Username,
			&comment.PostID, &comment.Likes, &comment.Dislikes,
			&comment.CreatedAt, &comment.UpdatedAt)
		if err != nil {
			continue
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

// Update modifies an existing comment
func (c *Comment) Update() error {
	// Validate comment content
	if err := c.Validate(); err != nil {
		return err
	}

	query := `
		UPDATE comments 
		SET content = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := database.GetDB().Exec(query, c.Content, now, c.ID)
	if err != nil {
		return err
	}

	c.UpdatedAt = now
	return nil
}

// Delete removes a comment from the database
func (c *Comment) Delete() error {
	// Start transaction for consistent deletion
	tx, err := database.GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all votes for this comment first
	_, err = tx.Exec("DELETE FROM votes WHERE comment_id = ?", c.ID)
	if err != nil {
		return err
	}

	// Delete the comment
	_, err = tx.Exec("DELETE FROM comments WHERE id = ?", c.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetCommentCount returns the total number of comments for a post
func GetCommentCount(postID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM comments WHERE post_id = ?`
	err := database.GetDB().QueryRow(query, postID).Scan(&count)
	return count, err
}

// UpdateVoteCounts recalculates and updates the like/dislike counts for a comment
func (c *Comment) UpdateVoteCounts() error {
	query := `
		UPDATE comments 
		SET likes = (SELECT COUNT(*) FROM votes WHERE comment_id = ? AND vote_type = 'like'),
		    dislikes = (SELECT COUNT(*) FROM votes WHERE comment_id = ? AND vote_type = 'dislike')
		WHERE id = ?
	`

	_, err := database.GetDB().Exec(query, c.ID, c.ID, c.ID)
	if err != nil {
		return err
	}

	// Refresh the counts in memory
	return c.refreshVoteCounts()
}

// refreshVoteCounts updates the in-memory vote counts
func (c *Comment) refreshVoteCounts() error {
	query := `SELECT likes, dislikes FROM comments WHERE id = ?`
	return database.GetDB().QueryRow(query, c.ID).Scan(&c.Likes, &c.Dislikes)
}

// getUserVote gets the current user's vote on this comment
func (c *Comment) getUserVote(userID int) {
	query := `SELECT vote_type FROM votes WHERE user_id = ? AND comment_id = ?`
	var voteType string
	err := database.GetDB().QueryRow(query, userID, c.ID).Scan(&voteType)
	if err == nil {
		c.UserVote = &voteType
	}
}

// Validate checks if comment content is valid
func (c *Comment) Validate() error {
	// Check content length
	if len(c.Content) == 0 {
		return errors.New("comment content cannot be empty")
	}
	if len(c.Content) > 1000 {
		return errors.New("comment content cannot exceed 1000 characters")
	}

	// Check for basic content (not just whitespace)
	trimmed := strings.TrimSpace(c.Content)
	if len(trimmed) == 0 {
		return errors.New("comment cannot be empty or contain only whitespace")
	}

	return nil
}

// CanEdit checks if a user can edit this comment
func (c *Comment) CanEdit(userID int) bool {
	return c.UserID == userID
}

// CanDelete checks if a user can delete this comment
func (c *Comment) CanDelete(userID int) bool {
	return c.UserID == userID
}
