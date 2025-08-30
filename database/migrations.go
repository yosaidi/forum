package database

import (
	"fmt"
	"log"
	"strings"
)

// RunMigrations creates all database tables and inserts default data
func RunMigrations() {
	log.Println("Running database migrations...")

	// Create tables in correct order
	createUsersTable()
	createCategoriesTable()
	createPostsTable()
	createCommentsTable()
	createVotesTable()
	createSessionsTable()

	AddUpdatedAtToCategories()

	log.Println("Database migrations completed successfully")
	fmt.Println()
}

// createUsersTable creates the users table for authentication
func createUsersTable() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		avatar VARCHAR(255) DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal("Failed to create users table:", err)
	}

	// Create indexes for performance on frequently queried columns
	createIndexIfNotExists("idx_users_username", "users", "username")
	createIndexIfNotExists("idx_users_email", "users", "email")

	log.Println("✓ Users table created")
}

// createCategoriesTable creates the categories table for forum sections
func createCategoriesTable() {
	query := `
	CREATE TABLE IF NOT EXISTS categories (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  name VARCHAR(50) UNIQUE NOT NULL,
	  description TEXT,
	  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal("Failed to create categories table:", err)
	}

	// Create index for category lookups
	createIndexIfNotExists("idx_categories_name", "categories", "name")

	// Insert default categories for the forum
	insertDefaultCategories()

	log.Println("✓ Categories table created")
}

// createPostsTable creates the posts table for forum discussions
func createPostsTable() {
	query := `
	CREATE TABLE IF NOt EXISTS posts(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title VARCHAR(255) NOT NULL,
	content TEXT NOT NULL,
	user_id INTEGER NOT NULL,
	category_id INTEGER NOT NULL,
	likes INTEGER DEFAULT 0,
    dislikes INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
	);`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal("Failed to create posts table:", err)
	}

	// Create indexes for performance
	createIndexIfNotExists("idx_posts_user_id", "posts", "user_id")
	createIndexIfNotExists("idx_posts_category_id", "posts", "category_id")
	createIndexIfNotExists("idx_posts_created_at", "posts", "created_at")

	log.Println("✓ Posts table created")
}

// createCommentsTable creates the comments table for post replies
func createCommentsTable() {
	query := `
	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		post_id INTEGER NOT NULL,
		likes INTEGER DEFAULT 0,
		dislikes INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
	);`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal("Failed to create comments table:", err)
	}

	// Create indexes for performance
	createIndexIfNotExists("idx_comments_user_id", "comments", "user_id")
	createIndexIfNotExists("idx_comments_post_id", "comments", "post_id")
	createIndexIfNotExists("idx_comments_created_at", "comments", "created_at")

	log.Println("✓ Comments table created")
}

func createVotesTable() {
	query := `
	CREATE TABLE IF NOT EXISTS votes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		post_id INTEGER,
		comment_id INTEGER,
		vote_type VARCHAR(10) NOT NULL CHECK(vote_type IN ('like', 'dislike')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE,
		UNIQUE(user_id, post_id, comment_id)
	);`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal("Failed to create votes table:", err)
	}

	// Create indexes for performance
	createIndexIfNotExists("idx_votes_user_id", "votes", "user_id")
	createIndexIfNotExists("idx_votes_post_id", "votes", "post_id")
	createIndexIfNotExists("idx_votes_comment_id", "votes", "comment_id")

	log.Println("✓ Votes table created")
}

// createSessionsTable creates the sessions table for user authentication
func createSessionsTable() {
	query := `
	CREATE TABLE IF NOT EXISTS sessions (
		id VARCHAR(255) PRIMARY KEY,
		user_id INTEGER NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal("Failed to create sessions table:", err)
	}

	// Create indexes for performance
	createIndexIfNotExists("idx_sessions_user_id", "sessions", "user_id")
	createIndexIfNotExists("idx_sessions_expires_at", "sessions", "expires_at")

	log.Println("✓ Sessions table created")
}

// createIndexIfNotExists creates an index only if it doesn't already exist
func createIndexIfNotExists(indexName, tableName, columnName string) {
	query := ` CREATE INDEX IF NOT EXISTS ` + indexName + ` ON ` + tableName + `(` + columnName + `);`
	if _, err := DB.Exec(query); err != nil {
		log.Printf("Warning: Failed to create index %s : %v", indexName, err)
	}
}

// insertDefaultCategories populates the categories table with default forum sections
func insertDefaultCategories() {
	categories := []struct {
		name        string
		description string
	}{
		{"general", "General Discussion - Talk about anything"},
		{"tech", "Technology - Latest tech news and discussions"},
		{"programming", "Programming - Code, languages, and development"},
		{"web-dev", "Web Development - Frontend, backend, and web technologies"},
		{"mobile", "Mobile Development - iOS, Android, and mobile apps"},
		{"career", "Career - Job advice, interviews, and career growth"},
		{"help", "Help & Support - Get help with your projects"},
		{"showcase", "Showcase - Show off your projects and creations"},
	}

	// Insert each category (INSERT OR IGNORE prevents duplicates)
	for _, cat := range categories {
		query := `INSERT OR IGNORE INTO categories (name, description) VALUES (?, ?)`
		_, err := DB.Exec(query, cat.name, cat.description)
		if err != nil {
			log.Printf("Warning: Failed to insert category %s: %v", cat.name, err)
		}
	}
	log.Println("✓ Default categories inserted")
}

// CleanExpiredSessions removes expired session records (call this periodically)
func CleanExpiredSessions() {
	query := `DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`
	result, err := DB.Exec(query)
	if err != nil {
		log.Printf("Warning: Failed to clean expired sessions: %v", err)
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned %d expired sessions", rowsAffected)
	}
}

// AddUpdatedAtToCategories adds updated_at column to existing categories table
func AddUpdatedAtToCategories() {
	// Check if updated_at column exists
	query := `SELECT sql FROM sqlite_master WHERE type='table' AND name='categories'`
	var tableSchema string
	err := DB.QueryRow(query).Scan(&tableSchema)
	if err != nil {
		log.Printf("Could not check categories table schema: %v", err)
		return
	}

	// If updated_at doesn't exist, add it
	if !strings.Contains(tableSchema, "updated_at") {
		alterQuery := `ALTER TABLE categories ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP`
		_, err := DB.Exec(alterQuery)
		if err != nil {
			log.Printf("Warning: Failed to add updated_at to categories: %v", err)
		} else {
			log.Println("✓ Added updated_at column to categories table")
		}
	}
}
