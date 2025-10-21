package models

import (
	"strings"
	"time"

	"forum/database"
)

type Post struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	UserID       int       `json:"user_id"`
	Username     string    `json:"username"`
	CategoryID   int       `json:"category_id"`
	CategoryName string    `json:"category_name"`
	Likes        int       `json:"likes"`
	Dislikes     int       `json:"dislikes"`
	CommentCount int       `json:"comment_count"`
	UserVote     *string   `json:"user_vote"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PostFilters struct {
	CurrentUserID int
	CategoryID    int
	AuthorID      int
	SortBy        string
	Limit         int
	Offset        int
}

func (p *Post) Create() error {
	query := `
		INSERT INTO posts (title, content, user_id, category_id, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := database.GetDB().Exec(query, p.Title, p.Content, p.UserID, p.CategoryID, now, now)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	p.ID = int(id)
	p.CreatedAt = now
	p.UpdatedAt = now

	return nil
}

func (p *Post) GetByID(id int, userID *int) error {
	query := `
		SELECT p.id, p.title, p.content, p.user_id, u.username, p.category_id, c.name,
        p.likes, p.dislikes, p.created_at, p.updated_at,
        (SELECT COUNT(*) FROM comments WHERE post_id = p.id) as comment_countS
   		FROM posts p
        JOIN users u ON p.user_id = u.id
        JOIN categories c ON p.category_id = c.id
        WHERE p.id = ?			    
	`

	row := database.DB.QueryRow(query, id)
	err := row.Scan(&p.ID, &p.Title, &p.Content, &p.UserID, &p.Username, &p.CategoryID,
		&p.CategoryName, &p.Likes, &p.Dislikes, &p.CreatedAt, &p.UpdatedAt, &p.CommentCount)
	if err != nil {
		return err
	}

	if userID != nil {
		p.GetUserVote(*userID)
	}

	return nil
}

func GetPosts(filters PostFilters) ([]Post, int, error) {
	var posts []Post
	var args []interface{}
	var whereClauses []string
	var joinClauses []string
	var orderClause string

	baseQuery := `
	SELECT 
		p.id, p.title, p.content, p.user_id, u.username, 
		p.category_id, c.name, p.likes, p.dislikes, 
		p.created_at, p.updated_at,
		(SELECT COUNT(*) FROM comments WHERE post_id = p.id) AS comment_count
	FROM posts p
	JOIN users u ON p.user_id = u.id
	JOIN categories c ON p.category_id = c.id
	`

	countQuery := `SELECT COUNT(*) FROM posts p
	JOIN users u ON p.user_id = u.id
	JOIN categories c ON p.category_id = c.id
	`

	// Filters
	if filters.CategoryID > 0 {
		whereClauses = append(whereClauses, "p.category_id = ?")
		args = append(args, filters.CategoryID)
	}
	if filters.AuthorID > 0 {
		whereClauses = append(whereClauses, "p.user_id = ?")
		args = append(args, filters.AuthorID)
	}

	// Special filters
	switch filters.SortBy {
	case "my_posts":
		if filters.CurrentUserID > 0 {
			whereClauses = append(whereClauses, "p.user_id = ?")
			args = append(args, filters.CurrentUserID)
		}
	case "my_likes":
		if filters.CurrentUserID > 0 {
			joinClauses = append(joinClauses, "JOIN votes v ON p.id = v.post_id AND v.user_id = ? AND v.vote_type = 'like'")
			args = append(args, filters.CurrentUserID)
		}
	case "my_dislikes":
		if filters.CurrentUserID > 0 {
			joinClauses = append(joinClauses, "JOIN votes v ON p.id = v.post_id AND v.user_id = ? AND v.vote_type = 'dislike'")
			args = append(args, filters.CurrentUserID)
		}
	}

	// Sorting
	switch filters.SortBy {
	case "oldest":
		orderClause = "ORDER BY p.created_at ASC"
	case "popular":
		orderClause = "ORDER BY p.likes DESC, p.dislikes ASC"
	default:
		orderClause = "ORDER BY p.created_at DESC"
	}

	// Assemble query parts in correct order
	if len(joinClauses) > 0 {
		baseQuery += " " + strings.Join(joinClauses, " ")
		countQuery += " " + strings.Join(joinClauses, " ")
	}
	if len(whereClauses) > 0 {
		where := " WHERE " + strings.Join(whereClauses, " AND ")
		baseQuery += where
		countQuery += where
	}
	baseQuery += " " + orderClause

	// Pagination
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, filters.Limit, filters.Offset)

	// Count
	var total int
	err := database.GetDB().QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		return posts, 0, err
	}

	// Execute posts query
	rows, err := database.GetDB().Query(baseQuery, args...)
	if err != nil {
		return posts, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID, &post.Title, &post.Content,
			&post.UserID, &post.Username,
			&post.CategoryID, &post.CategoryName,
			&post.Likes, &post.Dislikes,
			&post.CreatedAt, &post.UpdatedAt, &post.CommentCount,
		)
		if err != nil {
			continue
		}
		posts = append(posts, post)
	}

	return posts, total, nil
}

func (p *Post) GetUserVote(userID int) {
	query := `SELECT vote_type FROM votes WHERE user_id = ? AND post_id = ?`
	var voteType string
	err := database.GetDB().QueryRow(query, userID, p.ID).Scan(&voteType)
	if err == nil {
		p.UserVote = &voteType
	}
}

func (p *Post) GetVoteCounts() (int, int, error) {
	query := `
		SELECT 
			COUNT(CASE WHEN vote_type = 'like' THEN 1 END) AS likes,
			COUNT(CASE WHEN vote_type = 'dislike' THEN 1 END) AS dislikes
		FROM votes
		WHERE post_id = ?
	`
	var likes, dislikes int
	err := database.GetDB().QueryRow(query, p.ID).Scan(&likes, &dislikes)
	if err != nil {
		return 0, 0, err
	}
	p.Likes = likes
	p.Dislikes = dislikes
	return likes, dislikes, nil
}

func (p *Post) GetCommentCount() (int, error) {
	query := `SELECT COUNT(*) FROM comments WHERE post_id = ?`
	var count int
	err := database.GetDB().QueryRow(query, p.ID).Scan(&count)
	if err != nil {
		return 0, err
	}
	p.CommentCount = count
	return count, nil
}

func (p *Post) Update() error {
	query := `
		UPDATE posts 
		SET title = ?, content = ?, category_id = ?, updated_at = ? 
		WHERE id = ?
		`
	now := time.Now()
	_, err := database.GetDB().Exec(query, p.Title, p.Content, p.CategoryID, now, p.ID)
	if err != nil {
		return err
	}

	// Also update struct field so the controller response stays consistent
	p.UpdatedAt = now
	return nil
}

func (p *Post) Delete() error {
	// Start transaction for consistent deletion
	tx, err := database.GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete all votes for this post first
	_, err = tx.Exec("DELETE FROM votes WHERE post_id = ?", p.ID)
	if err != nil {
		return err
	}

	// Delete the post
	_, err = tx.Exec("DELETE FROM posts WHERE id = ?", p.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
