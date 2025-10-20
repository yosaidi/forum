package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	server := &http.Server{
		Addr:    config.GetPort(),
		Handler: mux,
	}

	go func() {
		log.Printf("Rate limiting is active - see configuration above")
		log.Printf("Server starting on http://localhost%s", config.GetPort())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
		fmt.Println()
	}()

	<-stop // Wait for interrupt signal
	log.Println("Received interrupt signal, shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	database.Close()
	log.Println("Server stopped")
}
