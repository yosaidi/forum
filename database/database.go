package database

import (
	"database/sql"
	"log"
	"time"

	"forum/config"

	_ "github.com/mattn/go-sqlite3"
)

// DB is the global database connection used with all models
var DB *sql.DB

// Init initializes the database connection and runs migrations
func Init() {
	var err error

	// Open connection to SQLite database
	DB, err = sql.Open("sqlite3", config.GetDatabaseURL())
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Test the database connection with Ping()
	if err = DB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Enable foreign key enforcement
	pragma := `PRAGMA foreign_keys = ON;`
	if _, err := DB.Exec(pragma); err != nil {
		log.Fatal("Failed to enable foreign key support:", err)
	}

	// Configure connection pool settings for better performance
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Database connected successfully")

	// Create all tables and insert default data
	RunMigrations()
}

// Close closes the database connection
func Close() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}

// GetDB returns the database connection
// so models and other packages may use this to perform operations
func GetDB() *sql.DB {
	return DB
}
