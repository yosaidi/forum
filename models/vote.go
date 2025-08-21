package models

import (
	"database/sql"
	"errors"
	"forum/database"
	"time"
)

// Vote represents a like or dislike on a post or comment
type Vote struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	PostID    *int      `json:"post_id"`    // NULL if voting on comment
	CommentID *int      `json:"comment_id"` // NULL if voting on post
	VoteType  string    `json:"vote_type"`  // "like" or "dislike"
	CreatedAt time.Time `json:"created_at"`
}

// VoteResult represents the result of a voting operation
type VoteResult struct {
	Action      string `json:"action"`       // "added", "removed", "changed"
	VoteType    string `json:"vote_type"`    // "like" or "dislike"
	NewLikes    int    `json:"new_likes"`    // Updated like count
	NewDislikes int    `json:"new_dislikes"` // Updated dislike count
}

// VoteStats represents voting statistics for a user
type VoteStats struct {
	TotalVotes    int `json:"total_votes"`
	LikesGiven    int `json:"likes_given"`
	DislikesGiven int `json:"dislikes_given"`
	LikesReceived int `json:"likes_received"`
}

// TogglePostVote handles voting logic for posts (like/dislike toggle)
func TogglePostVote(userID, postID int, voteType string) (*VoteResult, error) {
	// Validate vote type
	if voteType != "like" && voteType != "dislike" {
		return nil, errors.New("invalid vote type")
	}

	// Start transaction for data consistency
	tx, err := database.GetDB().Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Check if user already voted on this post
	var existingVoteType string
	query := `SELECT vote_type FROM votes WHERE user_id = ? AND post_id = ?`
	err = tx.QueryRow(query, userID, postID).Scan(&existingVoteType)

	var result VoteResult
	result.VoteType = voteType

	if err == sql.ErrNoRows {
		// No existing vote - create new vote
		err = createPostVote(tx, userID, postID, voteType)
		if err != nil {
			return nil, err
		}
		result.Action = "added"
	} else if err != nil {
		// Database error
		return nil, err
	} else if existingVoteType == voteType {
		// Same vote type - remove the vote (toggle off)
		err = removePostVote(tx, userID, postID)
		if err != nil {
			return nil, err
		}
		result.Action = "removed"
	} else {
		// Different vote type - update the vote
		err = updatePostVote(tx, userID, postID, voteType)
		if err != nil {
			return nil, err
		}
		result.Action = "changed"
	}

	// Update post vote counts
	err = updatePostVoteCounts(tx, postID)
	if err != nil {
		return nil, err
	}

	// Get updated counts
	result.NewLikes, result.NewDislikes, err = getPostVoteCounts(tx, postID)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// ToggleCommentVote handles voting logic for comments
func ToggleCommentVote(userID, commentID int, voteType string) (*VoteResult, error) {
	// Validate vote type
	if voteType != "like" && voteType != "dislike" {
		return nil, errors.New("invalid vote type")
	}

	// Start transaction for data consistency
	tx, err := database.GetDB().Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Check if user already voted on this comment
	var existingVoteType string
	query := `SELECT vote_type FROM votes WHERE user_id = ? AND comment_id = ?`
	err = tx.QueryRow(query, userID, commentID).Scan(&existingVoteType)

	var result VoteResult
	result.VoteType = voteType

	if err == sql.ErrNoRows {
		// No existing vote - create new vote
		err = createCommentVote(tx, userID, commentID, voteType)
		if err != nil {
			return nil, err
		}
		result.Action = "added"
	} else if err != nil {
		// Database error
		return nil, err
	} else if existingVoteType == voteType {
		// Same vote type - remove the vote (toggle off)
		err = removeCommentVote(tx, userID, commentID)
		if err != nil {
			return nil, err
		}
		result.Action = "removed"
	} else {
		// Different vote type - update the vote
		err = updateCommentVote(tx, userID, commentID, voteType)
		if err != nil {
			return nil, err
		}
		result.Action = "changed"
	}

	// Update comment vote counts
	err = updateCommentVoteCounts(tx, commentID)
	if err != nil {
		return nil, err
	}

	// Get updated counts
	result.NewLikes, result.NewDislikes, err = getCommentVoteCounts(tx, commentID)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetUserLikedPosts returns posts that a user has liked
func GetUserLikedPosts(userID int, limit, offset int) ([]Post, error) {
	var posts []Post

	query := `
		SELECT p.id, p.title, p.content, p.user_id, u.username, p.category_id, c.name,
		       p.likes, p.dislikes, p.created_at, p.updated_at,
		       (SELECT COUNT(*) FROM comments WHERE post_id = p.id) as comment_count
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN categories c ON p.category_id = c.id
		JOIN votes v ON p.id = v.post_id
		WHERE v.user_id = ? AND v.vote_type = 'like'
		ORDER BY v.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := database.GetDB().Query(query, userID, limit, offset)
	if err != nil {
		return posts, err
	}
	defer rows.Close()

	for rows.Next() {
		var post Post
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.UserID, &post.Username,
			&post.CategoryID, &post.CategoryName, &post.Likes, &post.Dislikes,
			&post.CreatedAt, &post.UpdatedAt, &post.CommentCount)
		if err != nil {
			continue
		}

		// User has liked this post by definition
		likeVote := "like"
		post.UserVote = &likeVote

		posts = append(posts, post)
	}

	return posts, nil
}

// GetVoteStats returns voting statistics for a user
func GetVoteStats(userID int) (*VoteStats, error) {
	stats := &VoteStats{}

	// Get votes given by user
	query := `
		SELECT 
			COUNT(*) as total_votes,
			SUM(CASE WHEN vote_type = 'like' THEN 1 ELSE 0 END) as likes_given,
			SUM(CASE WHEN vote_type = 'dislike' THEN 1 ELSE 0 END) as dislikes_given
		FROM votes 
		WHERE user_id = ?
	`
	err := database.GetDB().QueryRow(query, userID).Scan(&stats.TotalVotes, &stats.LikesGiven, &stats.DislikesGiven)
	if err != nil {
		return nil, err
	}

	// Get likes received on user's posts and comments
	query = `
		SELECT COUNT(*) FROM votes v
		LEFT JOIN posts p ON v.post_id = p.id
		LEFT JOIN comments c ON v.comment_id = c.id
		WHERE (p.user_id = ? OR c.user_id = ?) AND v.vote_type = 'like'
	`
	err = database.GetDB().QueryRow(query, userID, userID).Scan(&stats.LikesReceived)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// Helper functions for post voting
func createPostVote(tx *sql.Tx, userID, postID int, voteType string) error {
	query := `INSERT INTO votes (user_id, post_id, vote_type, created_at) VALUES (?, ?, ?, ?)`
	_, err := tx.Exec(query, userID, postID, voteType, time.Now())
	return err
}

func removePostVote(tx *sql.Tx, userID, postID int) error {
	query := `DELETE FROM votes WHERE user_id = ? AND post_id = ?`
	_, err := tx.Exec(query, userID, postID)
	return err
}

func updatePostVote(tx *sql.Tx, userID, postID int, voteType string) error {
	query := `UPDATE votes SET vote_type = ? WHERE user_id = ? AND post_id = ?`
	_, err := tx.Exec(query, voteType, userID, postID)
	return err
}

func updatePostVoteCounts(tx *sql.Tx, postID int) error {
	query := `
		UPDATE posts 
		SET likes = (SELECT COUNT(*) FROM votes WHERE post_id = ? AND vote_type = 'like'),
		    dislikes = (SELECT COUNT(*) FROM votes WHERE post_id = ? AND vote_type = 'dislike')
		WHERE id = ?
	`
	_, err := tx.Exec(query, postID, postID, postID)
	return err
}

func getPostVoteCounts(tx *sql.Tx, postID int) (int, int, error) {
	var likes, dislikes int
	query := `SELECT likes, dislikes FROM posts WHERE id = ?`
	err := tx.QueryRow(query, postID).Scan(&likes, &dislikes)
	return likes, dislikes, err
}

// Helper functions for comment voting
func createCommentVote(tx *sql.Tx, userID, commentID int, voteType string) error {
	query := `INSERT INTO votes (user_id, comment_id, vote_type, created_at) VALUES (?, ?, ?, ?)`
	_, err := tx.Exec(query, userID, commentID, voteType, time.Now())
	return err
}

func removeCommentVote(tx *sql.Tx, userID, commentID int) error {
	query := `DELETE FROM votes WHERE user_id = ? AND comment_id = ?`
	_, err := tx.Exec(query, userID, commentID)
	return err
}

func updateCommentVote(tx *sql.Tx, userID, commentID int, voteType string) error {
	query := `UPDATE votes SET vote_type = ? WHERE user_id = ? AND comment_id = ?`
	_, err := tx.Exec(query, voteType, userID, commentID)
	return err
}

func updateCommentVoteCounts(tx *sql.Tx, commentID int) error {
	query := `
		UPDATE comments 
		SET likes = (SELECT COUNT(*) FROM votes WHERE comment_id = ? AND vote_type = 'like'),
		    dislikes = (SELECT COUNT(*) FROM votes WHERE comment_id = ? AND vote_type = 'dislike')
		WHERE id = ?
	`
	_, err := tx.Exec(query, commentID, commentID, commentID)
	return err
}

func getCommentVoteCounts(tx *sql.Tx, commentID int) (int, int, error) {
	var likes, dislikes int
	query := `SELECT likes, dislikes FROM comments WHERE id = ?`
	err := tx.QueryRow(query, commentID).Scan(&likes, &dislikes)
	return likes, dislikes, err
}

// CleanupOrphanedVotes removes votes for deleted posts/comments
func CleanupOrphanedVotes() error {
	tx, err := database.GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove votes for deleted posts
	_, err = tx.Exec(`
		DELETE FROM votes 
		WHERE post_id IS NOT NULL 
		AND post_id NOT IN (SELECT id FROM posts)
	`)
	if err != nil {
		return err
	}

	// Remove votes for deleted comments
	_, err = tx.Exec(`
		DELETE FROM votes 
		WHERE comment_id IS NOT NULL 
		AND comment_id NOT IN (SELECT id FROM comments)
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}