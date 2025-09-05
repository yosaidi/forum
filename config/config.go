package config

import (
	"fmt"
	"log"
	"os"
)

// Config struct holds all application configuration settings
// and centralizes config management in one place
type Config struct {
	Port        string // HTTP servet port
	DatabaseURL string // Path to SQLite database file
}

// AppConfig is the global configuration instance
var AppConfig Config

// Load initializes the application configuration
func Load() {
	AppConfig = Config{
		Port:        getEnv("PORT", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "./database/forum.db"),
	}

	fmt.Println()
	log.Println("Configuration loaded")
	fmt.Println()
}

// GetPort returns the server port
func GetPort() string {
	return AppConfig.Port
}

// GetDatabaseURL returns the database file path
func GetDatabaseURL() string {
	return AppConfig.DatabaseURL
}

// getEnv is a helper function that reads environment variables
// If the environment variable doesn't exist, it returns the default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
