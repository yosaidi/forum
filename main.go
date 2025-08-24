package main

import (
	"fmt"
	"log"
	"net/http"

	"forum/config"
	"forum/database"
	"forum/routes"
)

func main() {
	// Load Config
	config.Load()

	// Initialize database (not used yet)
	database.Init()
	defer database.Close()

	// setup routes (later)
	mux := routes.SetupRoutes()

	// Optional: Print all routes for debugging
	fmt.Println("Available routes:")
	for _, route := range routes.GetRoutesList() {
		fmt.Printf("  %s\n", route)
	}

	// Start server
	log.Printf("Server starting on http://localhost%s", config.GetPort())
	if err := http.ListenAndServe(config.GetPort(), mux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
