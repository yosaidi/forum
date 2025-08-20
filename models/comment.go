package models

import (
	"database/sql"
	"time"

	"forum/database"
)

type Comment struct {
	ID        int       `json:"id"`
	Content   string    `json:"content"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	PostID    int       `json:"post_id"`
	Likes     int       `json:"likes"`
	Dislikes  int       `json:"dislikes"`
	UserVote  *string   `json:"user_vote"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *Comment) Create() error {
	query := `
	  INSERT INTO comments (content, user_id, post_id, created_at, updated_at)
	  VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := database.GetDB().Exec(query, c.Content, c.UserID, c.PostID, c.CreatedAt, c.UpdatedAt)
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
	c.Likes = 0
	c.Dislikes = 0
	return nil
}

func (c *Comment) GetByID(commentID int, userID *int) error {
	query := `
		SELECT c.id, c.content, c.user_id, u.username, c.post_id,
		       c.likes, c.dislikes, c.created_at, c.updated_at
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.id = ?
	`

	row := database.GetDB().QueryRow(query, commentID)
	err := row.Scan(&c.ID, &c.Content, &c.UserID, &c.Username, &c.PostID,
		&c.Likes, &c.Dislikes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return err
	}

	if userID != nil {
		c.getUserVote(*userID)
	}
	return nil
}

func getCommentsByPostID(postID int, userID *int, limit, offset int) ([]Comment, error) {
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
			&comment.PostID, &comment.Likes, &comment.Dislikes, &comment.CreatedAt, &comment.UpdatedAt)
		if err != nil {
			continue
		}

		if userID != nil {
			comment.getUserVote(*userID)
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (c *Comment) Update() error {
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

func (c *Comment) Delete() error {
	tx, err := database.GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM votes WHERE comment_id = ?", c.ID)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM comments WHERE id = ?", c.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func getCommentsCount(postID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM comments WHERE post_id = ?`
	err := database.GetDB().QueryRow(query, postID).Scan(&count)
	return count, err
}

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
			&comment.PostID, &comment.Likes, &comment.Dislikes, &comment.CreatedAt, &comment.UpdatedAt)
		if err != nil {
			continue
		}

		comment.UserVote = nil

		comments = append(comments, comment)
	}

	return comments, nil
}

func (c *Comment) CanEdit(userID int) bool {
	return c.UserID == userID
}

func (c *Comment) CanDelete(userID int) bool {
	// Maybe we could extend this to allow moderators/admins to delete any comment
	return c.UserID == userID
}

func (c *Comment) getUserVote(userID int) {
	query := `SELECT vote_type FROM votes WHERE user_id = ? AND comment_id = ?`
	var voteType string
	err := database.GetDB().QueryRow(query, userID, c.ID).Scan(&voteType)
	if err == nil {
		c.UserVote = &voteType
	}
	// If error (no vote found), UserVote remains nil
}

func (c *Comment) UpdateVoteCounts() error {
	tx, err := database.GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var likes int
	err = tx.QueryRow("SELECT COUNT(*) FROM votes WHERE comment_id = ? AND vote_type = 'like'", c.ID).Scan(&likes)
	if err != nil {
		return err
	}

	var dislikes int
	err = tx.QueryRow("SELECT COUNT(*) FROM votes WHERE comment_id = ? AND vote_type = 'dislike'", c.ID).Scan(&dislikes)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE comments SET likes = ?, dislikes = ? WHERE id = ?", likes, dislikes, c.ID)
	if err != nil {
		return err
	}

	c.Likes = likes
	c.Dislikes = dislikes

	return tx.Commit()
}

func (c *Comment) Validate() error {
	if len(c.Content) == 0 {
		return sql.ErrNoRows
	}
	if len(c.Content) > 1000 {
		return sql.ErrNoRows
	}
	if c.UserID <= 0 {
		return sql.ErrNoRows
	}
	if c.PostID <= 0 {
		return sql.ErrNoRows
	}
	return nil
}
