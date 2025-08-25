package routes

import (
	"net/http"
	"strconv"
	"strings"

	"forum/controllers"
	"forum/middleware"
)

// SetupRoutes configures all application routes using standard net/http
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// API routes with middleware wrapper
	mux.Handle("/api/", middleware.CORS(middleware.LogRequests(apiHandler())))

	// Static files (if needed)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./views/static/"))))

	// Serve SPA (index.html) for the root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./views/index.html")
	})

	return mux
}

// apiHandler returns the main API handler that routes all /api/* requests
func apiHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Remove /api prefix from path
		path := strings.TrimPrefix(r.URL.Path, "/api")

		// Route to appropriate handler based on path and method
		switch {
		// Authentication routes
		case path == "/auth/register" && r.Method == http.MethodPost:
			controllers.RegisterController(w, r)
		case path == "/auth/login" && r.Method == http.MethodPost:
			controllers.LoginController(w, r)
		case path == "/auth/logout" && r.Method == http.MethodPost:
			controllers.LogoutController(w, r)
		case path == "/auth/me" && r.Method == http.MethodGet:
			controllers.MeController(w, r)
		case path == "/auth/refresh" && r.Method == http.MethodPost:
			controllers.RefreshSessionController(w, r)
		case path == "/auth/check-username" && r.Method == http.MethodGet:
			controllers.CheckUsernameController(w, r)
		case path == "/auth/check-email" && r.Method == http.MethodGet:
			controllers.CheckEmailController(w, r)

		// Post routes
		case path == "/posts" && r.Method == http.MethodGet:
			middleware.OptionalAuth(controllers.GetPostsController)(w, r)
		case path == "/posts" && r.Method == http.MethodPost:
			middleware.RequireAuth(controllers.CreatePostController)(w, r)
		case matchPath(path, "/posts/", true) && r.Method == http.MethodGet:
			middleware.OptionalAuth(controllers.GetPostController)(w, r)
		case matchPath(path, "/posts/", true) && r.Method == http.MethodPut:
			middleware.RequireAuth(controllers.UpdatePostController)(w, r)
		case matchPath(path, "/posts/", true) && r.Method == http.MethodDelete:
			middleware.RequireAuth(controllers.DeletePostController)(w, r)
		case matchPath(path, "/posts/", true, "/vote") && r.Method == http.MethodPost:
			middleware.RequireAuth(controllers.VotePostController)(w, r)

		// Post comments routes
		case matchPath(path, "/posts/", true, "/comments") && r.Method == http.MethodGet:
			middleware.OptionalAuth(controllers.GetCommentsController)(w, r)
		case matchPath(path, "/posts/", true, "/comments") && r.Method == http.MethodPost:
			middleware.RequireAuth(controllers.CreateCommentController)(w, r)

		// Comment routes
		case path == "/comments" && r.Method == http.MethodPost:
			middleware.RequireAuth(controllers.CreateCommentController)(w, r)
		case matchPath(path, "/comments/", true) && r.Method == http.MethodGet:
			middleware.OptionalAuth(controllers.GetCommentController)(w, r)
		case matchPath(path, "/comments/", true) && r.Method == http.MethodPut:
			middleware.RequireAuth(controllers.UpdateCommentController)(w, r)
		case matchPath(path, "/comments/", true) && r.Method == http.MethodDelete:
			middleware.RequireAuth(controllers.DeleteCommentController)(w, r)
		case matchPath(path, "/comments/", true, "/vote") && r.Method == http.MethodPost:
			middleware.RequireAuth(controllers.VoteCommentController)(w, r)

		// User routes
		case matchPath(path, "/users/", true) && r.Method == http.MethodGet:
			controllers.GetUserProfileController(w, r)
		case matchPath(path, "/users/", true) && r.Method == http.MethodPut:
			middleware.RequireAuth(controllers.UpdateUserProfileController)(w, r)
		case matchPath(path, "/users/", true, "/posts") && r.Method == http.MethodGet:
			controllers.GetUserPostsController(w, r)
		case matchPath(path, "/users/", true, "/comments") && r.Method == http.MethodGet:
			controllers.GetUserCommentsController(w, r)
		case matchPath(path, "/users/", true, "/stats") && r.Method == http.MethodGet:
			controllers.GetUserStatsController(w, r)

		// Categories (if you implement this later)
		// case path == "/categories" && r.Method == http.MethodGet:
		//     controllers.GetCategoriesController(w, r)

		default:
			http.NotFound(w, r)
		}
	}
}

// Helper functions for URL matching

// matchPath checks if a path matches a pattern with optional ID and suffix
// Example: matchPath("/posts/123", "/posts/", true) returns true
// Example: matchPath("/posts/123/vote", "/posts/", true, "/vote") returns true
func matchPath(path, prefix string, expectID bool, suffix ...string) bool {
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	remaining := strings.TrimPrefix(path, prefix)

	if expectID {
		// Extract ID part
		parts := strings.Split(remaining, "/")
		if len(parts) == 0 {
			return false
		}

		// Check if first part is a number (ID)
		if _, err := strconv.Atoi(parts[0]); err != nil {
			return false
		}

		// If we have a suffix to match
		if len(suffix) > 0 {
			expectedSuffix := strings.Join(suffix, "")
			actualSuffix := strings.Join(parts[1:], "/")
			if actualSuffix != strings.TrimPrefix(expectedSuffix, "/") {
				return false
			}
		} else {
			// No suffix expected, should only be the ID
			if len(parts) > 1 {
				return false
			}
		}
	}

	return true
}

// extractIDFromPath extracts ID from URL path
// Example: extractIDFromPath("/api/posts/123") returns 123
func extractIDFromPath(path, prefix string) (int, error) {
	remaining := strings.TrimPrefix(path, prefix)
	parts := strings.FieldsFunc(remaining, func(r rune) bool { return r == '/' })
	if len(parts) == 0 {
		return 0, http.ErrNotSupported
	}

	return strconv.Atoi(parts[0])
}

// GetRoutesList returns a list of all available routes for debugging
func GetRoutesList() []string {
	return []string{
		"POST   /api/auth/register",
		"POST   /api/auth/login",
		"POST   /api/auth/logout",
		"GET    /api/auth/me",
		"POST   /api/auth/refresh",
		"GET    /api/auth/check-username",
		"GET    /api/auth/check-email",
		"",
		"GET    /api/posts",
		"POST   /api/posts",
		"GET    /api/posts/{id}",
		"PUT    /api/posts/{id}",
		"DELETE /api/posts/{id}",
		"POST   /api/posts/{id}/vote",
		"",
		"GET    /api/posts/{id}/comments",
		"POST   /api/posts/{id}/comments",
		"POST   /api/comments",
		"GET    /api/comments/{id}",
		"PUT    /api/comments/{id}",
		"DELETE /api/comments/{id}",
		"POST   /api/comments/{id}/vote",
		"",
		"GET    /api/users/{id}",
		"PUT    /api/users/{id}",
		"GET    /api/users/{id}/posts",
		"GET    /api/users/{id}/comments",
		"GET    /api/users/{id}/stats",
	}
}

// Alternative simpler setup if you prefer a more basic approach
func SetupSimpleRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Apply CORS and logging to all API routes
	apiWithMiddleware := func(pattern string, handler http.HandlerFunc) {
		mux.HandleFunc(pattern, middleware.CORS(middleware.LogRequests(handler)))
	}

	// Auth routes
	apiWithMiddleware("/api/auth/register", controllers.RegisterController)
	apiWithMiddleware("/api/auth/login", controllers.LoginController)
	apiWithMiddleware("/api/auth/logout", controllers.LogoutController)
	apiWithMiddleware("/api/auth/me", controllers.MeController)

	// Post routes (public)
	apiWithMiddleware("/api/posts", handlePosts)
	apiWithMiddleware("/api/posts/", handlePostByID)

	// Comment routes
	apiWithMiddleware("/api/comments", handleComments)
	apiWithMiddleware("/api/comments/", handleCommentByID)

	// User routes
	apiWithMiddleware("/api/users/", handleUsers)

	return mux
}

// Handler functions for the simple approach

func handlePosts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		middleware.OptionalAuth(controllers.GetPostsController)(w, r)
	case http.MethodPost:
		middleware.RequireAuth(controllers.CreatePostController)(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handlePostByID(w http.ResponseWriter, r *http.Request) {
	// Check if this is a sub-resource (comments, vote)
	path := r.URL.Path
	if strings.Contains(path, "/comments") {
		switch r.Method {
		case http.MethodGet:
			middleware.OptionalAuth(controllers.GetCommentsController)(w, r)
		case http.MethodPost:
			middleware.RequireAuth(controllers.CreateCommentController)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.Contains(path, "/vote") {
		if r.Method == http.MethodPost {
			middleware.RequireAuth(controllers.VotePostController)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle main post operations
	switch r.Method {
	case http.MethodGet:
		middleware.OptionalAuth(controllers.GetPostController)(w, r)
	case http.MethodPut:
		middleware.RequireAuth(controllers.UpdatePostController)(w, r)
	case http.MethodDelete:
		middleware.RequireAuth(controllers.DeletePostController)(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleComments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		middleware.RequireAuth(controllers.CreateCommentController)(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCommentByID(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(r.URL.Path, "/vote") {
		if r.Method == http.MethodPost {
			middleware.RequireAuth(controllers.VoteCommentController)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		middleware.OptionalAuth(controllers.GetCommentController)(w, r)
	case http.MethodPut:
		middleware.RequireAuth(controllers.UpdateCommentController)(w, r)
	case http.MethodDelete:
		middleware.RequireAuth(controllers.DeleteCommentController)(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if strings.Contains(path, "/posts") {
		controllers.GetUserPostsController(w, r)
		return
	}

	if strings.Contains(path, "/comments") {
		controllers.GetUserCommentsController(w, r)
		return
	}

	if strings.Contains(path, "/stats") {
		controllers.GetUserStatsController(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		controllers.GetUserProfileController(w, r)
	case http.MethodPut:
		middleware.RequireAuth(controllers.UpdateUserProfileController)(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
