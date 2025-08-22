package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/middleware"
	"forum/models"
	"forum/utils"
)

// PostCreateRequest represents the JSON structure for creating posts
type PostCreateRequest struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	CategoryID int    `json:"category_id"`
}

// PostUpdateRequest represents the JSON structure for updating posts
type PostUpdateRequest struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	CategoryID int    `json:"category_id"`
}

// PostResponse represents post data sent to client
type PostResponse struct {
	ID           int          `json:"id"`
	Title        string       `json:"title"`
	Content      string       `json:"content"`
	CategoryID   int          `json:"category_id"`
	Category     string       `json:"category"`
	Author       UserResponse `json:"author"`
	LikeCount    int          `json:"like_count"`
	DislikeCount int          `json:"dislike_count"`
	CommentCount int          `json:"comment_count"`
	UserVote     *string      `json:"user_vote"` // "like", "dislike", or null
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// CreatePostController handles post creation
func CreatePostController(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.MethodNotAllowed(w, "Only POST method allowed")
		return
	}

	// Get authenticated user (requires middleware)
	userID, exists := middleware.GetUserIDFromContext(r)
	if !exists {
		utils.Unauthorized(w, "Authentication required")
		return
	}

	// Parse JSON request body
	var req PostCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	// Validate post form
	if errors := utils.ValidatePostForm(req.Title, req.Content, req.CategoryID); errors.HasErrors() {
		utils.ValidationError(w, errors)
		return
	}

	// Create new post
	post := models.Post{
		Title:      req.Title,
		Content:    req.Content,
		CategoryID: req.CategoryID,
		UserID:     userID,
	}

	if err := post.Create(); err != nil {
		utils.InternalServerError(w, "Failed to create post")
		return
	}

	// Get full post details for responses
	postResponse, err := getPostResponse(&post, userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve post details")
		return
	}
	utils.Created(w, "Post created successfully", postResponse)
}

// GetPostController handles retrieving a single post
func GetPostController(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	// Get post ID from URL path
	postID, err := getPostIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid post ID")
		return
	}

	// Get current user ID if available (for vote info)
	userID, _ := middleware.GetUserIDFromContext(r)
	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	// Get post from database
	post := models.Post{}
	if err := post.GetByID(postID, userIDPtr); err != nil {
		utils.NotFound(w, "Post not found")
		return
	}

	// Get full post details for response
	postResponse, err := getPostResponse(&post, userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve post details")
		return
	}

	utils.Success(w, "Post retrieved successfully", postResponse)
}

// UpdatePostController handles post updates
func UpdatePostController(w http.ResponseWriter, r *http.Request) {
	// Only allow PUT requests
	if r.Method != http.MethodPut {
		utils.MethodNotAllowed(w, "Only PUT method allowed")
		return
	}

	// Get authenticated user
	userID, exists := middleware.GetUserIDFromContext(r)
	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	if !exists {
		utils.Unauthorized(w, "Authentication required")
		return
	}

	// Get post ID from URL path
	postID, err := getPostIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid post ID")
		return
	}

	post := models.Post{}
	if err := post.GetByID(postID, userIDPtr); err != nil {
		utils.NotFound(w, "Post not found")
		return
	}

	// Check ownership
	if post.UserID != userID {
		utils.Forbidden(w, "You can only edit your own posts")
		return
	}

	// Parse JSON request body
	var req PostUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	// Validate post form
	if errors := utils.ValidatePostForm(req.Title, req.Content, req.CategoryID); errors.HasErrors() {
		utils.ValidationError(w, errors)
		return
	}

	// Update post fields
	post.Title = req.Title
	post.Content = req.Content
	post.CategoryID = req.CategoryID

	if err := post.Update(); err != nil {
		utils.InternalServerError(w, "Failed to update post")
		return
	}

	// Get full post details for response
	postResponse, err := getPostResponse(&post, userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve updated post details")
		return
	}

	utils.Success(w, "Post updated successfully", postResponse)
}

// DeletePostController handles post deletion
func DeletePostController(w http.ResponseWriter, r *http.Request) {
	// Only allow DELETE requests
	if r.Method != http.MethodDelete {
		utils.MethodNotAllowed(w, "Only DELETE method allowed")
		return
	}

	// Get authenticated user
	userID, exists := middleware.GetUserIDFromContext(r)
	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	if !exists {
		utils.Unauthorized(w, "Authentication required")
		return
	}

	// Get post ID from URL path
	postID, err := getPostIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid post ID")
		return
	}

	// Get existing post
	post := models.Post{}
	if err := post.GetByID(postID, userIDPtr); err != nil {
		utils.NotFound(w, "Post not found")
		return
	}

	// Check ownership
	if post.UserID != userID {
		utils.Forbidden(w, "You can only delete your own posts")
		return
	}

	// Delete post
	if err := post.Delete(); err != nil {
		utils.InternalServerError(w, "Failed to delete post")
		return
	}

	utils.Success(w, "Post deleted successfully", nil)
}

// GetPostsController handles retrieving multiple posts with filtering
func GetPostsController(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	// Parse query parameters
	query := r.URL.Query()

	// Pagination
	page, _ := strconv.Atoi(query.Get("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	// Validate pagination
	limit, offset, err := utils.ValidatePagination(page, limit)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Filters
	categoryID, _ := strconv.Atoi(query.Get("category"))
	authorID, _ := strconv.Atoi(query.Get("author"))
	sortBy := query.Get("sort") // "newest", "oldest", "popular"

	if sortBy == "" {
		sortBy = "newest"
	}

	// Get current user ID if available (for vote info)
	userID, _ := middleware.GetUserIDFromContext(r)

	// Get posts from database
	posts, total, err := models.GetPosts(models.PostFilters{
		CategoryID: categoryID,
		AuthorID:   authorID,
		SortBy:     sortBy,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve posts")
		return
	}

	// Convert to response format
	var postResponses []PostResponse
	for _, post := range posts {
		postResponse, err := getPostResponse(&post, userID)
		if err != nil {
			utils.InternalServerError(w, "Failed to process post data")
			return
		}
		postResponses = append(postResponses, *postResponse)
	}

	// Prepare pagination info
	pagination := map[string]interface{}{
		"current_page": page,
		"per_page":     limit,
		"total":        total,
		"total_pages":  (total + limit - 1) / limit,
		"has_next":     page < (total+limit-1)/limit,
		"has_prev":     page > 1,
	}

	utils.PaginatedSuccess(w, "Posts retrieved successfully", postResponses, pagination)
}

// VotePostController handles post voting (like/dislike)
func VotePostController(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		utils.MethodNotAllowed(w, "Only POST method allowed")
		return
	}

	// Get authenticated user
	userID, exists := middleware.GetUserIDFromContext(r)
	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	if !exists {
		utils.Unauthorized(w, "Authentication required")
		return
	}

	// Get post ID from URL path
	postID, err := getPostIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid post ID")
		return
	}

	// Parse vote type from request body
	var voteData struct {
		VoteType string `json:"vote_type"` // "like" or "dislike"
	}
	if err := json.NewDecoder(r.Body).Decode(&voteData); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	// Validate vote type
	if !utils.IsValidVoteType(voteData.VoteType) {
		utils.BadRequest(w, "Vote type must be 'like' or 'dislike'")
		return
	}

	// Check if post exists
	post := models.Post{}
	if err := post.GetByID(postID, userIDPtr); err != nil {
		utils.NotFound(w, "Post not found")
		return
	}

	// Create vote
	result, err := models.TogglePostVote(userID, postID, voteData.VoteType) // This handles like/dislike toggle logic
	if err != nil {
		utils.InternalServerError(w, "Failed to process vote")
		return
	}

	utils.Success(w, "Vote processed successfully", map[string]interface{}{
		"action":        result.Action,   // "added", "removed", "changed"
		"vote_type":     result.VoteType, // current vote type or null
		"like_count":    result.NewLikes,
		"dislike_count": result.NewDislikes,
	})
}

// Helper functions

// getPostIDFromPath extracts post ID from URL path like /posts/123
func getPostIDFromPath(path string) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return 0, errors.New("invalid path format")
	}

	return strconv.Atoi(parts[1])
}

// getPostResponse converts a Post model to PostResponse with additional data
func getPostResponse(post *models.Post, currentUserID int) (*PostResponse, error) {
	// Get author info
	author := models.User{}
	if err := author.GetByID(post.UserID); err != nil {
		return nil, err
	}

	// Get vote counts
	likeCount, dislikeCount, err := post.GetVoteCounts()
	if err != nil {
		return nil, err
	}

	// Get comment count
	commentCount, err := post.GetCommentCount()
	if err != nil {
		return nil, err
	}

	// Get user's vote if logged in
	var userVote *string
	if currentUserID > 0 {
		vote := models.Vote{}
		if err := vote.GetByUserAndPost(currentUserID, post.ID); err == nil {
			userVote = &vote.VoteType
		}
	}

	// Get category name (you might need to implement this)
	categoryName := "General" // Default or fetch from categories table

	return &PostResponse{
		ID:         post.ID,
		Title:      post.Title,
		Content:    post.Content,
		CategoryID: post.CategoryID,
		Category:   categoryName,
		Author: UserResponse{
			ID:       author.ID,
			Username: author.Username,
			Email:    author.Email,
			JoinedAt: author.CreatedAt,
		},
		LikeCount:    likeCount,
		DislikeCount: dislikeCount,
		CommentCount: commentCount,
		UserVote:     userVote,
		CreatedAt:    post.CreatedAt,
		UpdatedAt:    post.UpdatedAt,
	}, nil
}
