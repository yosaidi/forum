package models

import (
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

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PostCount   int    `json:"post_count"`
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
		p.getUserVote(*userID)
	}

	return nil
}

func GetAllPosts(userID *int, categoryFilter string, limit, offset int) ([]Post, error) {
	var posts []Post
	var query string
	var args []interface{}

	baseQuery := `
        SELECT p.id, p.title, p.content, p.user_id, u.username, p.category_id, c.name,
               p.likes, p.dislikes, p.created_at, p.updated_at,
               (SELECT COUNT(*) FROM comments WHERE post_id = p.id) as comment_count
        FROM posts p
        JOIN users u ON p.user_id = u.id
        JOIN categories c ON p.category_id = c.id
    `

	if categoryFilter != "" && categoryFilter != "all" {
		query = baseQuery + " WHERE c.name = ? ORDER BY p.created_at DESC LIMIT ? OFFSET ?"
		args = []interface{}{categoryFilter, limit, offset}
	} else {
		query = baseQuery + " ORDER BY p.created_at DESC LIMIT ? OFFSET ?"
		args = []interface{}{limit, offset}
	}

	rows, err := database.GetDB().Query(query, args...)
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

		if userID != nil {
			post.getUserVote(*userID)
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func (p *Post) getUserVote(userID int) {
	query := `SELECT vote_type FROM votes WHERE user_id = ? AND post_id = ?`
	var voteType string
	err := database.GetDB().QueryRow(query, userID, p.ID).Scan(&voteType)
	if err == nil {
		p.UserVote = &voteType
	}
}

func GetAllCategories() ([]Category, error) {
	var categories []Category

	query := `
	    SELECT c.id, c.name, c.description, COUNT(p.id) as post_count
        FROM categories c
        LEFT JOIN posts p ON c.id = p.category_id
        GROUP BY c.id, c.name, c.description
        ORDER BY c.name
	`

	rows, err := database.GetDB().Query(query)
	if err != nil {
		return categories, nil
	}
	defer rows.Close()

	for rows.Next() {
		var category Category
		err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.PostCount)
		if err != nil {
			continue
		}
		categories = append(categories, category)
	}
	return categories, nil
}
