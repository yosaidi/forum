package routes

import (
	"net/http"
	"strconv"
	"strings"

	"forum/controllers"
	"forum/middleware"
)

// Constant file paths
const (
	StaticDir  = "./views/static/"
	UploadsDir = "./uploads/"
	IndexFile  = "./views/index.html"
)

// SetupRoutes configures all application routes using standard net/http
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// API routes with global middleware applied once
	apiHandlerWithMiddlware := middleware.RateLimit(
		middleware.LogRequests(
			middleware.OptionalAuth(apiHandler()),
		),
	)
	mux.Handle("/api/", apiHandlerWithMiddlware)

	// Static files (CSS, JS, images, etc.)
	mux.Handle("/static/",
		http.StripPrefix("/static/",
			http.FileServer(
				http.Dir(StaticDir)),
		),
	)

	// Uploaded files (avatars, etc.)
	mux.Handle("/uploads/",
		http.StripPrefix("/uploads/",
			http.FileServer(http.Dir(UploadsDir)),
		),
	)

	// SPA fallback for all other routes (except API & static)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") ||
			strings.HasPrefix(r.URL.Path, "/static/") ||
			strings.HasPrefix(r.URL.Path, "/uploads/") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, IndexFile)
	})

	return mux
}

// Route defines a single API route
type Route struct {
	Method       string
	Path         string // Uses {id} as placeholder for numeric IDs
	Handler      http.HandlerFunc
	RequiresAuth bool
}

// List of all API routes
var apiRoutes = []Route{
	// Auth routes
	{Method: http.MethodPost, Path: "/auth/register", Handler: controllers.RegisterController},
	{Method: http.MethodPost, Path: "/auth/login", Handler: controllers.LoginController},
	{Method: http.MethodPost, Path: "/auth/logout", Handler: controllers.LogoutController, RequiresAuth: true},
	{Method: http.MethodGet, Path: "/auth/me", Handler: controllers.MeController, RequiresAuth: true},
	{Method: http.MethodPost, Path: "/auth/refresh", Handler: controllers.RefreshSessionController},
	{Method: http.MethodGet, Path: "/auth/check-username", Handler: controllers.CheckUsernameController},
	{Method: http.MethodGet, Path: "/auth/check-email", Handler: controllers.CheckEmailController},

	// Post routes
	{Method: http.MethodGet, Path: "/posts", Handler: middleware.OptionalAuth(controllers.GetPostsController)},
	{Method: http.MethodPost, Path: "/posts", Handler: middleware.RequireAuth(controllers.CreatePostController), RequiresAuth: true},
	{Method: http.MethodGet, Path: "/posts/{id}", Handler: middleware.OptionalAuth(controllers.GetPostController)},
	{Method: http.MethodPut, Path: "/posts/{id}", Handler: middleware.RequireAuth(controllers.UpdatePostController), RequiresAuth: true},
	{Method: http.MethodDelete, Path: "/posts/{id}", Handler: middleware.RequireAuth(controllers.DeletePostController), RequiresAuth: true},
	{Method: http.MethodPost, Path: "/posts/{id}/vote", Handler: middleware.RequireAuth(controllers.VotePostController), RequiresAuth: true},

	// Post comments
	{Method: http.MethodGet, Path: "/posts/{id}/comments", Handler: middleware.OptionalAuth(controllers.GetCommentsController)},
	{Method: http.MethodPost, Path: "/posts/{id}/comments", Handler: middleware.RequireAuth(controllers.CreateCommentController), RequiresAuth: true},

	// Comments
	{Method: http.MethodPost, Path: "/comments", Handler: middleware.RequireAuth(controllers.CreateCommentController), RequiresAuth: true},
	{Method: http.MethodGet, Path: "/comments/{id}", Handler: middleware.OptionalAuth(controllers.GetCommentController)},
	{Method: http.MethodPut, Path: "/comments/{id}", Handler: middleware.RequireAuth(controllers.UpdateCommentController), RequiresAuth: true},
	{Method: http.MethodDelete, Path: "/comments/{id}", Handler: middleware.RequireAuth(controllers.DeleteCommentController), RequiresAuth: true},
	{Method: http.MethodPost, Path: "/comments/{id}/vote", Handler: middleware.RequireAuth(controllers.VoteCommentController), RequiresAuth: true},

	// Users
	{Method: http.MethodGet, Path: "/users/{id}", Handler: controllers.GetUserProfileController},
	{Method: http.MethodPut, Path: "/users/{id}", Handler: middleware.RequireAuth(controllers.UpdateUserProfileController), RequiresAuth: true},
	{Method: http.MethodPost, Path: "/users/{id}/avatar", Handler: middleware.RequireAuth(controllers.UploadAvatarController), RequiresAuth: true},
	{Method: http.MethodDelete, Path: "/users/{id}/avatar", Handler: middleware.RequireAuth(controllers.DeleteAvatarController), RequiresAuth: true},
	{Method: http.MethodGet, Path: "/users/{id}/posts", Handler: controllers.GetUserPostsController},
	{Method: http.MethodGet, Path: "/users/{id}/comments", Handler: controllers.GetUserCommentsController},
	{Method: http.MethodGet, Path: "/users/{id}/stats", Handler: controllers.GetUserStatsController},

	// Categories
	{Method: http.MethodGet, Path: "/categories", Handler: controllers.GetCategoriesController},
	{Method: http.MethodGet, Path: "/categories/{id}", Handler: controllers.GetCategoryController},
}

// apiHandler returns the main API handler that routes all /api/* requests
func apiHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api")

		for _, route := range apiRoutes {
			if r.Method != route.Method {
				continue
			}

			// Match paths, including {id} placeholders
			if matchRoute(path, route.Path) {
				handler := route.Handler
				if route.RequiresAuth {
					handler = middleware.RequireAuth(handler)
				}
				handler(w, r)
				return
			}
		}

		http.NotFound(w, r)
	}
}

// matchRoute matches dynamic paths with {id} placeholder
func matchRoute(actual, template string) bool {
	actualParts := strings.Split(strings.Trim(actual, "/"), "/")
	templateParts := strings.Split(strings.Trim(template, "/"), "/")

	if len(actualParts) != len(templateParts) {
		return false
	}

	for i := 0; i < len(templateParts); i++ {
		if templateParts[i] == "{id}" {
			// We need to ensure {id} is a number
			if _, err := strconv.Atoi(actualParts[i]); err != nil {
				return false
			}
			continue
		}
		if templateParts[i] != actualParts[i] {
			return false
		}
	}
	return true
}

// GetRoutesList returns a list of all available routes for debugging
func GetRoutesList() []string {
	return []string{
		// Auth routes
		"POST   /api/auth/register",
		"POST   /api/auth/login",
		"POST   /api/auth/logout",
		"GET    /api/auth/me",
		"POST   /api/auth/refresh",
		"GET    /api/auth/check-username",
		"GET    /api/auth/check-email",
		"",

		// Post routes
		"GET    /api/posts",
		"POST   /api/posts",
		"GET    /api/posts/{id}",
		"PUT    /api/posts/{id}",
		"DELETE /api/posts/{id}",
		"POST   /api/posts/{id}/vote",
		"",
		"GET    /api/posts/{id}/comments",
		"POST   /api/posts/{id}/comments",
		"",

		// Comment routes
		"POST   /api/comments",
		"GET    /api/comments/{id}",
		"PUT    /api/comments/{id}",
		"DELETE /api/comments/{id}",
		"POST   /api/comments/{id}/vote",
		"",

		// User routes
		"GET    /api/users/{id}",
		"PUT    /api/users/{id}",
		"POST   /api/users/{id}/avatar",
		"DELETE /api/users/{id}/avatar",
		"GET    /api/users/{id}/posts",
		"GET    /api/users/{id}/comments",
		"GET    /api/users/{id}/stats",
		"",

		// Category routes
		"GET    /api/categories",
		"GET    /api/categories/{id}",
		"",

		// Static & Uploads (optional)
		"GET    /static/*",
		"GET    /uploads/*",
		"",
	}
}
