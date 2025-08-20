package config

import (
	"log"
	"os"
)

// Config struct holds all application configuration settings
// and centralizes config management in one place
type Config struct {
	Port        string // HTTP servet port
	DatabaseURL string // Path to SQLite database file
	SecretKey   string // Secret key for session encryption and security
	Environment string // App environment (dev, prod, test)
}

// AppConfig is the global configuration instance
var AppConfig Config


// Load initializes the application configuration
func Load() {
	AppConfig = Config{
		Port:        getEnv("PORT", ":8080"),
		DatabaseURL: getEnv("DATABASE_URL", "./database/forum.db"),
		SecretKey:   getEnv("SECRET_KEY", "your-secret-key-change-this-in-production"), // IMPORTANT: to change this in production!
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	log.Println("Configuration loaded")
}

// GetPort returns the server port
func GetPort() string {
	return AppConfig.Port
}

// GetDatabaseURL returns the database file path
func GetDatabaseURL() string {
	return AppConfig.DatabaseURL
}

// GetSecretKey returns the secret key for encryption
func GetSecretKey() string {
	return AppConfig.SecretKey
}

// IsDevelopment checks if we're running in development mode
func IsDevelopment() bool {
	return AppConfig.Environment == "development"
}

// getEnv is a helper function that reads environment variables
// If the environment variable doesn't exist, it returns the default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
