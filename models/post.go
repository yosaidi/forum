package models

import (
	"errors"
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
	CategoryID int
	AuthorID   int
	SortBy     string
	Limit      int
	Offset     int
}

// type Category struct {
// 	ID          int    `json:"id"`
// 	Name        string `json:"name"`
// 	Description string `json:"description"`
// 	PostCount   int    `json:"post_count"`
// }

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

	baseQuery := `
	SELECT p.id, p.title, p.content, p.user_id, u.username, p.category_id, c.name,
	p.likes, p.dislikes, p.created_at, p.updated_at,
	(SELECT COUNT(*) FROM comments WHERE post_id = p.id) as comment_count
	FROM posts p
	JOIN users u ON p.user_id = u.id
	JOIN categories c ON p.category_id = c.id
    `

	countQuery := "SELECT COUNT(*) FROM posts p"

	// Filters
	if filters.CategoryID > 0 {
		whereClauses = append(whereClauses, "p.category_id = ?")
		args = append(args, filters.CategoryID)
	}
	if filters.AuthorID > 0 {
		whereClauses = append(whereClauses, "p.user_id = ?")
		args = append(args, filters.AuthorID)
	}

	if len(whereClauses) > 0 {
		where := " WHERE " + strings.Join(whereClauses, " AND ")
		baseQuery += where
		countQuery += where
	}

	// Sorting
	switch filters.SortBy {
	case "oldest":
		baseQuery += " ORDER BY p.created_at ASC"
	case "popular":
		baseQuery += " ORDER BY p.likes DESC, p.dislikes ASC"
	default: // newest
		baseQuery += " ORDER BY p.created_at DESC"
	}

	// Pagination
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, filters.Limit, filters.Offset)

	// Get total count
	var total int
	err := database.GetDB().QueryRow(countQuery, args[:len(args)-2]...).Scan(&total) // exclude limit/offset
	if err != nil {
		return posts, 0, err
	}

	// Get posts
	rows, err := database.GetDB().Query(baseQuery, args...)
	if err != nil {
		return posts, 0, err
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


// 		var category Category
// 		err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.PostCount)
// 		if err != nil {
// 			continue
// 		}
// 		categories = append(categories, category)
// 	}
// 	return categories, nil
// }

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
	query := `DELETE FROM posts WHERE id = ?`
	res, err := database.GetDB().Exec(query, p.ID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("no post deleted")
	}

	return nil
}

// func GetAllPosts(userID *int, categoryFilter string, limit, offset int) ([]Post, error) {
// 	var posts []Post
// 	var query string
// 	var args []interface{}

// 	baseQuery := `
// 		SELECT p.id, p.title, p.content, p.user_id, u.username, p.category_id, c.name,
// 			   p.likes, p.dislikes, p.created_at, p.updated_at,
// 			   (SELECT COUNT(*) FROM comments WHERE post_id = p.id) as comment_count
// 		FROM posts p
// 		JOIN users u ON p.user_id = u.id
// 		JOIN categories c ON p.category_id = c.id
// 	`

// 	if categoryFilter != "" && categoryFilter != "all" {
// 		query = baseQuery + " WHERE c.name = ? ORDER BY p.created_at DESC LIMIT ? OFFSET ?"
// 		args = []interface{}{categoryFilter, limit, offset}
// 	} else {
// 		query = baseQuery + " ORDER BY p.created_at DESC LIMIT ? OFFSET ?"
// 		args = []interface{}{limit, offset}
// 	}

// 	rows, err := database.GetDB().Query(query, args...)
// 	if err != nil {
// 		return posts, err
// 	}
// 	defer rows.Close()

// 	for rows.Next() {
// 		var post Post
// 		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.UserID, &post.Username,
// 			&post.CategoryID, &post.CategoryName, &post.Likes, &post.Dislikes,
// 			&post.CreatedAt, &post.UpdatedAt, &post.CommentCount)
// 		if err != nil {
// 			continue
// 		}

// 		if userID != nil {
// 			post.GetUserVote(*userID)
// 		}

// 		posts = append(posts, post)
// 	}

// 	return posts, nil
// }
