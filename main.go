package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"forum/config"
	"forum/database"
	"forum/middleware"
	"forum/routes"
)

func main() {
	// Load Config
	config.Load()

	// Initialize database
	database.Init()

	// Setup custom rate limits for your forum
	middleware.SetupCustomRateLimits()

	// Setup routes with enhanced rate limiting
	mux := routes.SetupRoutes()

	// Print rate limit configuration
	middleware.PrintRateLimitConfig()

	// Print all routes for debugging
	fmt.Println("Available routes:")
	fmt.Println()
	for _, route := range routes.GetRoutesList() {
		fmt.Printf("  %s\n", route)
	}

	// Handling OS signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Start server in a goroutine
	go func() {
		log.Printf("Rate limiting is active - see configuration above")
		log.Printf("Server starting on http://localhost%s", config.GetPort())
		if err := http.ListenAndServe(config.GetPort(), mux); err != nil {
			log.Fatal("Server failed to start:", err)
		}
		fmt.Println()
	}()

	<-stop // Wait for interrupt signal

	fmt.Println()
	log.Println("Shutting down gracefully...")
	database.Close()
	log.Println("Server stopped")
}
