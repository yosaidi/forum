package main

import (
	"log"
	"net/http"

	"forum/config"
	"forum/database"
)

func main() {
	// Load Config
	config.Load()

	// Initialize database (not used yet)
	database.Init()
	defer database.Close()

	// setup routes (later)
	// router :=

	var router http.Handler
	// Start server
	log.Printf("Server starting on http://localshost%s", config.GetPort())
	log.Fatal(http.ListenAndServe(config.GetPort(), router))
}
