package middleware

import (
	"net/http"

	"forum/config"
)

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		setCORSHeaders(w)

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Continue to next handler
		next(w, r)
	}
}

// setCORSHeaders sets the necessary CORS headers
func setCORSHeaders(w http.ResponseWriter) {
	// Allow specific origins or all origins in development
	if config.IsDevelopment() {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		// In production, specify exact origins
		w.Header().Set("Access-Control-Allow-Origin", "http//localhost:8080")
	}

	// Allowed HTTP methods
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

	// Allowed headers
	w.Header().Set("Access-Control-Allow-Headers",
		"Content-Type, Authorization, X-Requested-With, Accept")

	// Allow credentials (cookies)
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	// Cache preflight response for 24 hours
	w.Header().Set("Access-Control-Max-Age", "86400")

	// Expose custom headers to frontend
	w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count, X-Page-Count")
}
