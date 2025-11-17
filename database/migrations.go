package database

import (
	"fmt"
	"log"
)

// RunMigrations creates all database tables and inserts default data
func RunMigrations() {
	log.Println("Running database migrations...")

	// Create tables in correct order
	createUsersTable()
	createCategoriesTable()
	createPostsTable()
	createCommentsTable()
	createPostCategoriesTable() // does order matter ?
	migratePostsToMultipleCategories()

	createVotesTable()
	createSessionsTable()

	log.Println("Database migrations completed successfully")
	fmt.Println()
}



// createUsersTable creates the users table for authentication
func createUsersTable() {
	// Users table creation
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
	// Catergories table creation
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
	// Posts table creation with foreign keys to users and categories
	query := `
	CREATE TABLE IF NOT EXISTS posts(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title VARCHAR(255) NOT NULL,
	content TEXT NOT NULL,
	user_id INTEGER NOT NULL,
	--category_id INTEGER NOT NULL,
	likes INTEGER DEFAULT 0,
    dislikes INTEGER DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal("Failed to create posts table:", err)
	}

	// Create indexes for performance
	createIndexIfNotExists("idx_posts_user_id", "posts", "user_id")
	//createIndexIfNotExists("idx_posts_category_id", "posts", "category_id")
	createIndexIfNotExists("idx_posts_created_at", "posts", "created_at")

	log.Println("✓ Posts table created")
}

// createCommentsTable creates the comments table for post replies
func createCommentsTable() {
	// Comments table creation with foreign keys to users and posts
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

// added categories table
// createCommentsTable creates the comments table for post replies
func createPostCategoriesTable() {
	// Comments table creation with foreign keys to users and posts
	query := `
	CREATE TABLE IF NOT EXISTS post_categories (
    post_id INTEGER NOT NULL,
    category_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, category_id),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
	);
	`

	if _, err := DB.Exec(query); err != nil {
		log.Fatal("Failed to create post categories table:", err)
	}

	// Create indexes for performance
	createIndexIfNotExists("idx_post_categories_post_id", "post_categories", "post_id")
	createIndexIfNotExists("idx_post_categories_category_id", "post_categories", "category_id")

	log.Println("✓ post_categories table created")
}

// migration is crucial

func migratePostsToMultipleCategories() {
	log.Println("Starting migration: posts.category_id -> post_categories table...")
	
	// Step 1: Check if category_id column exists
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('posts') WHERE name='category_id'").Scan(&count)
	if err != nil {
		log.Printf("Warning: Could not check for category_id column: %v", err)
		return
	}
	
	if count == 0 {
		log.Println("✓ No category_id column found - migration not needed")
		return
	}
	
	log.Println("  → Found category_id column, migrating data...")
	
	// Step 2: Move existing data to junction table
	result, err := DB.Exec(`
		INSERT INTO post_categories (post_id, category_id)
		SELECT id, category_id FROM posts WHERE category_id IS NOT NULL
	`)
	if err != nil {
		log.Printf("Warning: Failed to migrate category data: %v", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		log.Printf("  → Migrated %d post-category relationships", rowsAffected)
	}
	
	// Step 3: Drop the category_id column (requires table recreation in SQLite)
	log.Println("  → Recreating posts table without category_id column...")
	
	// Begin transaction for safety
	tx, err := DB.Begin()
	if err != nil {
		log.Printf("Warning: Could not start transaction: %v", err)
		return
	}
	defer tx.Rollback() // Will be ignored if we commit successfully
	
	// Create new table without category_id
	_, err = tx.Exec(`
		CREATE TABLE posts_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		log.Printf("Error: Failed to create new posts table: %v", err)
		return
	}
	
	// Copy data from old table to new (excluding category_id)
	_, err = tx.Exec(`
		INSERT INTO posts_new (id, title, content, user_id, created_at, updated_at)
		SELECT id, title, content, user_id, created_at, updated_at
		FROM posts
	`)
	if err != nil {
		log.Printf("Error: Failed to copy post data: %v", err)
		return
	}
	
	// Drop old table
	_, err = tx.Exec(`DROP TABLE posts`)
	if err != nil {
		log.Printf("Error: Failed to drop old posts table: %v", err)
		return
	}
	
	// Rename new table to posts
	_, err = tx.Exec(`ALTER TABLE posts_new RENAME TO posts`)
	if err != nil {
		log.Printf("Error: Failed to rename new posts table: %v", err)
		return
	}
	
	// Recreate indexes
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_posts_user_id ON posts(user_id)`)
	if err != nil {
		log.Printf("Warning: Failed to create user_id index: %v", err)
	}
	
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at)`)
	if err != nil {
		log.Printf("Warning: Failed to create created_at index: %v", err)
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error: Failed to commit migration: %v", err)
		return
	}
	
	log.Println("✓ Successfully migrated posts table - category_id column removed")
}

func createVotesTable() {
	// Votes table creation with foreign keys to users, posts, and comments
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
	// Sessions table creation with foreign key to users
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
