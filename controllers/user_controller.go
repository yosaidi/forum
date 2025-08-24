package controllers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/database"
	"forum/models"
	"forum/utils"
)

// UserProfile represents the public user profile data
type UserProfile struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Avatar       string    `json:"avatar"`
	CreatedAt    time.Time `json:"created_at"`
	PostCount    int       `json:"post_count"`
	CommentCount int       `json:"comment_count"`
}

// UserStats represents detailed user statistics
type UserStats struct {
	UserProfile
	TotalPostLikes    int `json:"total_post_likes"`
	TotalCommentLikes int `json:"total_comment_likes"`
	AccountAge        int `json:"account_age_days"`
}

// UserUpdateRequest represents the request body for updating user profile
type UserUpdateRequest struct {
	Avatar string `json:"avatar"`
}

// GetUserProfileController handles GET /api/users/{id}
func GetUserProfileController(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.GetIDFromURL(r, "/users/")
	if err != nil {
		utils.BadRequest(w, "Invalid user ID")
		return
	}

	// Get user from database
	var user models.User
	err = user.GetByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.NotFound(w, "User not found")
			return
		}
		utils.InternalServerError(w, "Failed to get user")
		return
	}

	// Get user stats
	postCount, err := getUserPostCount(userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to get user stats")
		return
	}

	commentCount, err := getUserCommentCount(userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to get user stats")
		return
	}

	// Create public profile response
	profile := UserProfile{
		ID:           user.ID,
		Username:     user.Username,
		Avatar:       user.GetAvatarURL(),
		CreatedAt:    user.CreatedAt,
		PostCount:    postCount,
		CommentCount: commentCount,
	}

	utils.Success(w, "User profile retrieved successfully", profile)
}

// UpdateUserProfileController handles PUT /api/users/{id}
func UpdateUserProfileController(w http.ResponseWriter, r *http.Request) {
	// Get current user from session
	currentUser, err := GetCurrentUser(r)
	if err != nil {
		utils.Unauthorized(w, "Authentication required")
		return
	}

	userID, err := utils.GetIDFromURL(r, "/users/")
	if err != nil {
		utils.BadRequest(w, "Invalid user ID")
		return
	}

	// Check if user can only update their own profile
	if currentUser.ID != userID {
		utils.Forbidden(w, "You can only update your own profile")
		return
	}

	// Parse request body
	var updateReq UserUpdateRequest
	if err := utils.ParseJSON(r, &updateReq); err != nil {
		utils.BadRequest(w, "Invalid request body")
		return
	}

	// Update avatar if provided
	if updateReq.Avatar != "" {
		err = currentUser.UpdateAvatar(updateReq.Avatar)
		if err != nil {
			utils.InternalServerError(w, "Failed to update avatar")
			return
		}
	}

	// Return updated profile
	profile := UserProfile{
		ID:        currentUser.ID,
		Username:  currentUser.Username,
		Avatar:    currentUser.GetAvatarURL(),
		CreatedAt: currentUser.CreatedAt,
	}

	utils.Success(w, "Profile updated successfully", profile)
}

// GetUserPostsController handles GET /api/users/{id}/posts
func GetUserPostsController(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.GetIDFromURL(r, "/users/")
	if err != nil {
		utils.BadRequest(w, "Invalid user ID")
		return
	}

	// Parse query parameters for pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Verify user exists
	var user models.User
	err = user.GetByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.NotFound(w, "User not found")
			return
		}
		utils.InternalServerError(w, "Failed to get user")
		return
	}

	// Get user's posts with pagination
	posts, err := getUserPosts(userID, limit, offset)
	if err != nil {
		utils.InternalServerError(w, "Failed to get user posts")
		return
	}

	// Get total count for pagination info
	totalPosts, err := getUserPostCount(userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to get post count")
		return
	}

	response := map[string]interface{}{
		"posts":       posts,
		"page":        page,
		"limit":       limit,
		"total":       totalPosts,
		"total_pages": (totalPosts + limit - 1) / limit,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"avatar":   user.GetAvatarURL(),
		},
	}

	utils.Success(w, "User posts retrieved successfully", response)
}

// GetUserCommentsController handles GET /api/users/{id}/comments
func GetUserCommentsController(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.GetIDFromURL(r, "/users/")
	if err != nil {
		utils.BadRequest(w, "Invalid user ID")
		return
	}

	// Parse query parameters for pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Verify user exists
	var user models.User
	err = user.GetByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.NotFound(w, "User not found")
			return
		}
		utils.InternalServerError(w, "Failed to get user")
		return
	}

	// Get user's comments with pagination
	comments, err := getUserComments(userID, limit, offset)
	if err != nil {
		utils.InternalServerError(w, "Failed to get user comments")
		return
	}

	// Get total count for pagination info
	totalComments, err := getUserCommentCount(userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to get comment count")
		return
	}

	response := map[string]interface{}{
		"comments":    comments,
		"page":        page,
		"limit":       limit,
		"total":       totalComments,
		"total_pages": (totalComments + limit - 1) / limit,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"avatar":   user.GetAvatarURL(),
		},
	}

	utils.Success(w, "User comments retrieved successfully", response)
}

// GetUserStatsController handles GET /api/users/{id}/stats
func GetUserStatsController(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.GetIDFromURL(r, "/users/")
	if err != nil {
		utils.BadRequest(w, "Invalid user ID")
		return
	}

	// Get user from database
	var user models.User
	err = user.GetByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.NotFound(w, "User not found")
			return
		}
		utils.InternalServerError(w, "Failed to get user")
		return
	}

	// Get detailed stats
	postCount, err := getUserPostCount(userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to get post count")
		return
	}

	commentCount, err := getUserCommentCount(userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to get comment count")
		return
	}

	postLikes, err := getUserPostLikes(userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to get post likes")
		return
	}

	commentLikes, err := getUserCommentLikes(userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to get comment likes")
		return
	}

	accountAge := int(time.Since(user.CreatedAt).Hours() / 24)

	// Create detailed stats response
	stats := UserStats{
		UserProfile: UserProfile{
			ID:           user.ID,
			Username:     user.Username,
			Avatar:       user.GetAvatarURL(),
			CreatedAt:    user.CreatedAt,
			PostCount:    postCount,
			CommentCount: commentCount,
		},
		TotalPostLikes:    postLikes,
		TotalCommentLikes: commentLikes,
		AccountAge:        accountAge,
	}

	utils.Success(w, "User stats retrieved successfully", stats)
}

// GetCurrentUser gets the current user from session and returns the User model
func GetCurrentUser(r *http.Request) (*models.User, error) {
	userID, err := utils.GetUserIDFromSession(r)
	if err != nil {
		return nil, err
	}

	var user models.User
	err = user.GetByID(userID)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Helper functions

func getUserPostCount(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM posts WHERE author_id = ?`
	var count int
	err := database.GetDB().QueryRow(query, userID).Scan(&count)
	return count, err
}

func getUserCommentCount(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM comments WHERE author_id = ?`
	var count int
	err := database.GetDB().QueryRow(query, userID).Scan(&count)
	return count, err
}

func getUserPostLikes(userID int) (int, error) {
	query := `
		SELECT COALESCE(SUM(CASE WHEN v.vote_type = 'up' THEN 1 WHEN v.vote_type = 'down' THEN -1 ELSE 0 END), 0)
		FROM posts p
		LEFT JOIN votes v ON p.id = v.post_id AND v.comment_id IS NULL
		WHERE p.author_id = ?
	`
	var likes int
	err := database.GetDB().QueryRow(query, userID).Scan(&likes)
	return likes, err
}

func getUserCommentLikes(userID int) (int, error) {
	query := `
		SELECT COALESCE(SUM(CASE WHEN v.vote_type = 'up' THEN 1 WHEN v.vote_type = 'down' THEN -1 ELSE 0 END), 0)
		FROM comments c
		LEFT JOIN votes v ON c.id = v.comment_id
		WHERE c.author_id = ?
	`
	var likes int
	err := database.GetDB().QueryRow(query, userID).Scan(&likes)
	return likes, err
}

func getUserPosts(userID, limit, offset int) ([]map[string]interface{}, error) {
	query := `
		SELECT p.id, p.title, p.content, p.category, p.created_at, p.updated_at,
			   u.username, u.avatar,
			   COALESCE(SUM(CASE WHEN v.vote_type = 'up' THEN 1 WHEN v.vote_type = 'down' THEN -1 ELSE 0 END), 0) as vote_score,
			   COUNT(DISTINCT c.id) as comment_count
		FROM posts p
		LEFT JOIN users u ON p.author_id = u.id
		LEFT JOIN votes v ON p.id = v.post_id AND v.comment_id IS NULL
		LEFT JOIN comments c ON p.id = c.post_id
		WHERE p.author_id = ?
		GROUP BY p.id, p.title, p.content, p.category, p.created_at, p.updated_at, u.username, u.avatar
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := database.GetDB().Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []map[string]interface{}
	for rows.Next() {
		var id, voteScore, commentCount int
		var title, content, category, username, avatar string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &title, &content, &category, &createdAt, &updatedAt,
			&username, &avatar, &voteScore, &commentCount)
		if err != nil {
			return nil, err
		}

		// Truncate content for list view
		truncatedContent := content
		if len(content) > 200 {
			truncatedContent = content[:200] + "..."
		}

		post := map[string]interface{}{
			"id":            id,
			"title":         title,
			"content":       truncatedContent,
			"category":      category,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
			"author":        username,
			"author_avatar": strings.TrimSpace(avatar),
			"vote_score":    voteScore,
			"comment_count": commentCount,
		}

		if post["author_avatar"] == "" {
			post["author_avatar"] = "/static/avatars/default.png"
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func getUserComments(userID, limit, offset int) ([]map[string]interface{}, error) {
	query := `
		SELECT c.id, c.content, c.created_at, c.updated_at, c.post_id,
			   p.title as post_title,
			   u.username, u.avatar,
			   COALESCE(SUM(CASE WHEN v.vote_type = 'up' THEN 1 WHEN v.vote_type = 'down' THEN -1 ELSE 0 END), 0) as vote_score
		FROM comments c
		LEFT JOIN posts p ON c.post_id = p.id
		LEFT JOIN users u ON c.author_id = u.id
		LEFT JOIN votes v ON c.id = v.comment_id
		WHERE c.author_id = ?
		GROUP BY c.id, c.content, c.created_at, c.updated_at, c.post_id, p.title, u.username, u.avatar
		ORDER BY c.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := database.GetDB().Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []map[string]interface{}
	for rows.Next() {
		var id, postID, voteScore int
		var content, postTitle, username, avatar string
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &content, &createdAt, &updatedAt, &postID,
			&postTitle, &username, &avatar, &voteScore)
		if err != nil {
			return nil, err
		}

		// Truncate content for list view
		truncatedContent := content
		if len(content) > 150 {
			truncatedContent = content[:150] + "..."
		}

		comment := map[string]interface{}{
			"id":            id,
			"content":       truncatedContent,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
			"post_id":       postID,
			"post_title":    postTitle,
			"author":        username,
			"author_avatar": strings.TrimSpace(avatar),
			"vote_score":    voteScore,
		}

		if comment["author_avatar"] == "" {
			comment["author_avatar"] = "/static/avatars/default.png"
		}

		comments = append(comments, comment)
	}

	return comments, nil
}
