package controllers

import (
	"database/sql"
	"fmt"
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
	Email        string    `json:"email,omitempty"`
	Avatar       string    `json:"avatar"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
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
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
}

// GetUserProfileController handles GET /api/users/{id}
func GetUserProfileController(w http.ResponseWriter, r *http.Request) {
	userID, err := utils.GetIDFromURL(r, "/users/")
	if err != nil {
		utils.BadRequest(w, "Invalid user ID")
		return
	}

	// Get current user (if authenticated)
	currentUser, _ := GetCurrentUser(r)
	isOwnProfile := currentUser != nil && currentUser.ID == userID

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
		UpdatedAt:    user.UpdatedAt,
		PostCount:    postCount,
		CommentCount: commentCount,
	}

	// Add email only for profile owner
	if isOwnProfile {
		profile.Email = user.Email
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

	// Update fields if provided
	updated := false

	// Update username if provided
	if updateReq.Username != "" && updateReq.Username != currentUser.Username {
		if err := utils.ValidateUsername(updateReq.Username); err != nil {
			utils.BadRequest(w, "Invalid username: "+err.Error())
			return
		}

		// Check if username is already taken
		var existingUser models.User
		err := existingUser.GetByUsername(updateReq.Username)
		if err == nil {
			utils.Conflict(w, "Username already taken")
			return
		} else if err != sql.ErrNoRows {
			utils.InternalServerError(w, "Failed to check username availability")
			return
		}

		if err := currentUser.UpdateUsername(updateReq.Username); err != nil {
			utils.InternalServerError(w, "Failed to update username")
			return
		}
		updated = true
	}

	// Update email if provided
	if updateReq.Email != "" && updateReq.Email != currentUser.Email {
		if err := utils.ValidateEmail(updateReq.Email); err != nil {
			utils.BadRequest(w, "Invalid email: "+err.Error())
			return
		}

		// Check if email is already taken
		var existingUser models.User
		err := existingUser.GetByEmail(updateReq.Email)
		if err == nil {
			utils.Conflict(w, "Email already taken")
			return
		} else if err != sql.ErrNoRows {
			utils.InternalServerError(w, "Failed to check email availability")
			return
		}

		if err := currentUser.UpdateEmail(updateReq.Email); err != nil {
			utils.InternalServerError(w, "Failed to update email")
			return
		}
		updated = true
	}

	// Update avatar if provided
	if updateReq.Avatar != "" {
		err = currentUser.UpdateAvatar(updateReq.Avatar)
		if err != nil {
			utils.InternalServerError(w, "Failed to update avatar")
			return
		}

		updated = true
	}

	if !updated {
		utils.BadRequest(w, "No valid fields provided for update")
		return
	}

	// Return updated profile
	postCount, _ := getUserPostCount(currentUser.ID)
	commentCount, _ := getUserCommentCount(currentUser.ID)

	// Return updated profile
	profile := UserProfile{
		ID:           currentUser.ID,
		Username:     currentUser.Username,
		Email:        currentUser.Email,
		Avatar:       currentUser.GetAvatarURL(),
		CreatedAt:    currentUser.CreatedAt,
		UpdatedAt:    currentUser.UpdatedAt,
		PostCount:    postCount,
		CommentCount: commentCount,
	}

	utils.Success(w, "Profile updated successfully", profile)
}

// UploadAvatarController handles POST /api/users/{id}/avatar
func UploadAvatarController(w http.ResponseWriter, r *http.Request) {
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

	// Check if user can only update their own avatar
	if currentUser.ID != userID {
		utils.Forbidden(w, "You can only update your own avatar")
		return
	}

	// Handle file upload
	uploadResult, err := utils.HandleFileUpload(r, "avatar", utils.AvatarUploadConfig)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Delete old avatar file if it exists and is not default
	if currentUser.Avatar != "" && !strings.Contains(currentUser.Avatar, "default.png") {
		oldFilename := utils.ExtractFilenameFromURL(currentUser.Avatar)
		if oldFilename != "" {
			oldFilePath := utils.GetAvatarFilePath(oldFilename)
			utils.DeleteFile(oldFilePath)
		}
	}

	// Update user avatar in database
	err = currentUser.UpdateAvatar(uploadResult.URL)
	if err != nil {
		// If database update fails, remove uploaded file
		utils.DeleteFile(utils.GetAvatarFilePath(uploadResult.Filename))
		utils.InternalServerError(w, "Failed to update avatar")
		return
	}

	// Return upload result
	response := map[string]interface{}{
		"avatar_url": uploadResult.URL,
		"file_info":  uploadResult,
	}

	utils.Success(w, "Avatar uploaded successfully", response)
}

// DeleteAvatarController handles DELETE /api/users/{id}/avatar
func DeleteAvatarController(w http.ResponseWriter, r *http.Request) {
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

	// Check if user can only delete their own avatar
	if currentUser.ID != userID {
		utils.Forbidden(w, "You can only delete your own avatar")
		return
	}

	// Delete current avatar file if it exists and is not default
	if currentUser.Avatar != "" && !strings.Contains(currentUser.Avatar, "default.png") {
		filename := utils.ExtractFilenameFromURL(currentUser.Avatar)
		if filename != "" {
			filePath := utils.GetAvatarFilePath(filename)
			utils.DeleteFile(filePath)
		}
	}

	// Reset avatar to default in database
	err = currentUser.UpdateAvatar("")
	if err != nil {
		utils.InternalServerError(w, "Failed to reset avatar")
		return
	}

	utils.Success(w, "Avatar deleted successfully", map[string]string{
		"avatar_url": currentUser.GetAvatarURL(),
	})
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
			UpdatedAt:    user.UpdatedAt,
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
	query := `SELECT COUNT(*) FROM posts WHERE user_id = ?`
	var count int
	err := database.GetDB().QueryRow(query, userID).Scan(&count)
	return count, err
}

func getUserCommentCount(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM comments WHERE user_id = ?`
	var count int
	err := database.GetDB().QueryRow(query, userID).Scan(&count)
	return count, err
}

func getUserPostLikes(userID int) (int, error) {
	query := `
		SELECT COALESCE(SUM(CASE WHEN v.vote_type = 'like' THEN 1 WHEN v.vote_type = 'dislike' THEN -1 ELSE 0 END), 0)
		FROM posts p
		LEFT JOIN votes v ON p.id = v.post_id AND v.comment_id IS NULL
		WHERE p.user_id = ?
	`
	var likes int
	err := database.GetDB().QueryRow(query, userID).Scan(&likes)
	return likes, err
}

func getUserCommentLikes(userID int) (int, error) {
	query := `
		SELECT COALESCE(SUM(CASE WHEN v.vote_type = 'like' THEN 1 WHEN v.vote_type = 'dislike' THEN -1 ELSE 0 END), 0)
		FROM comments c
		LEFT JOIN votes v ON c.id = v.comment_id
		WHERE c.user_id = ?
	`
	var likes int
	err := database.GetDB().QueryRow(query, userID).Scan(&likes)
	return likes, err
}

func getUserPosts(userID, limit, offset int) ([]map[string]interface{}, error) {
	query := `
		SELECT p.id, p.title, p.content, p.created_at, p.updated_at,
		       u.username, u.avatar,
		       COALESCE(SUM(CASE WHEN v.vote_type = 'like' THEN 1 WHEN v.vote_type = 'dislike' THEN -1 ELSE 0 END), 0) as vote_score,
		       COUNT(DISTINCT c.id) as comment_count
		FROM posts p
		LEFT JOIN users u ON p.user_id = u.id
		LEFT JOIN votes v ON p.id = v.post_id AND v.comment_id IS NULL
		LEFT JOIN comments c ON p.id = c.post_id
		WHERE p.user_id = ?
		GROUP BY p.id, p.title, p.content, p.created_at, p.updated_at, u.username, u.avatar
		ORDER BY p.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := database.GetDB().Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var posts []map[string]interface{}
	for rows.Next() {
		var id, voteScore, commentCount int
		var title, content, username string
		var avatar sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(&id, &title, &content, &createdAt, &updatedAt,
			&username, &avatar, &voteScore, &commentCount)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		truncatedContent := content
		if len(content) > 200 {
			truncatedContent = content[:200] + "..."
		}

		// Get categories for this post
		categoryQuery := `
			SELECT c.id, c.name
			FROM categories c
			INNER JOIN post_categories pc ON c.id = pc.category_id
			WHERE pc.post_id = ?
		`

		categoryRows, err := database.GetDB().Query(categoryQuery, id)
		if err != nil {
			return nil, fmt.Errorf("category query error: %w", err)
		}

		var categories []map[string]interface{}
		for categoryRows.Next() {
			var catID int
			var catName string
			if err := categoryRows.Scan(&catID, &catName); err != nil {
				categoryRows.Close()
				return nil, fmt.Errorf("category scan error: %w", err)
			}
			categories = append(categories, map[string]interface{}{
				"id":   catID,
				"name": catName,
			})
		}
		categoryRows.Close()

		avatarStr := ""
		if avatar.Valid {
			avatarStr = strings.TrimSpace(avatar.String)
		}

		post := map[string]interface{}{
			"id":            id,
			"title":         title,
			"content":       truncatedContent,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
			"author":        username,
			"author_avatar": avatarStr,
			"vote_score":    voteScore,
			"comment_count": commentCount,
			"categories":    categories,
		}

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return posts, nil
}

func getUserComments(userID, limit, offset int) ([]map[string]interface{}, error) {
	query := `
		SELECT c.id, c.content, c.created_at, c.updated_at, c.post_id,
			   p.title as post_title,
			   u.username, u.avatar,
			   COALESCE(SUM(CASE WHEN v.vote_type = 'like' THEN 1 WHEN v.vote_type = 'dislike' THEN -1 ELSE 0 END), 0) as vote_score
		FROM comments c
		LEFT JOIN posts p ON c.post_id = p.id
		LEFT JOIN users u ON c.user_id = u.id
		LEFT JOIN votes v ON c.id = v.comment_id
		WHERE c.user_id = ?
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
