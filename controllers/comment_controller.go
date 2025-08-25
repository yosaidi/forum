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

// CommentCreateRequest represents the JSON structure for creating comments
type CommentCreateRequest struct {
	Content string `json:"content"`
	PostID  int    `json:"post_id"`
}

// CommentUpdateRequest represents the JSON structure for updating comments
type CommentUpdateRequest struct {
	Content string `json:"content"`
}

// CommentResponse represents comment data sent to client
type CommentResponse struct {
	ID        int          `json:"id"`
	Content   string       `json:"content"`
	PostID    int          `json:"post_id"`
	Author    UserResponse `json:"author"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// CreateCommentController handles comment creation
func CreateCommentController(w http.ResponseWriter, r *http.Request) {
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

	// Parse JSON request body
	var req CommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	// Validate comment form
	if errors := utils.ValidateCommentForm(req.Content); errors.HasErrors() {
		utils.ValidationError(w, errors)
		return
	}

	// Validate post ID
	if err := utils.ValidateID(req.PostID, "post_id"); err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	// Check if post exists
	post := models.Post{}
	if err := post.GetByID(req.PostID, userIDPtr); err != nil {
		utils.BadRequest(w, "Post not found")
		return
	}

	// Create new comment
	comment := models.Comment{
		Content: req.Content,
		PostID:  req.PostID,
		UserID:  userID,
	}

	if err := comment.Create(); err != nil {
		utils.InternalServerError(w, "Failed to create comment")
		return
	}

	// Get full comment details for response
	commentResponse, err := getCommentResponse(&comment)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve comment details")
		return
	}

	utils.Created(w, "Comment created successfully", commentResponse)
}

// GetCommentController handles retrieving a single comment
func GetCommentController(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	// Get authenticated user
	userID, _ := middleware.GetUserIDFromContext(r)
	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	// Get comment ID from URL path
	commentID, err := getCommentIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid comment ID")
		return
	}

	// Get comment from database
	comment := models.Comment{}
	if err := comment.GetByID(commentID, userIDPtr); err != nil {
		utils.NotFound(w, "Comment not found")
		return
	}

	// Get full comment details for response
	commentResponse, err := getCommentResponse(&comment)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve comment details")
		return
	}

	utils.Success(w, "Comment retrieved successfully", commentResponse)
}

// UpdateCommentController handles comment updates
func UpdateCommentController(w http.ResponseWriter, r *http.Request) {
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

	// Get comment ID from URL path
	commentID, err := getCommentIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid comment ID")
		return
	}

	// Get existing comment
	comment := models.Comment{}
	if err := comment.GetByID(commentID, userIDPtr); err != nil {
		utils.NotFound(w, "Comment not found")
		return
	}

	// Check ownership
	if comment.UserID != userID {
		utils.Forbidden(w, "You can only edit your own comments")
		return
	}

	// Parse JSON request body
	var req CommentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	// Validate comment form
	if errors := utils.ValidateCommentForm(req.Content); errors.HasErrors() {
		utils.ValidationError(w, errors)
		return
	}

	// Update comment content
	comment.Content = req.Content

	if err := comment.Update(); err != nil {
		utils.InternalServerError(w, "Failed to update comment")
		return
	}

	// Get full comment details for response
	commentResponse, err := getCommentResponse(&comment)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve updated comment details")
		return
	}

	utils.Success(w, "Comment updated successfully", commentResponse)
}

// DeleteCommentController handles comment deletion
func DeleteCommentController(w http.ResponseWriter, r *http.Request) {
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

	// Get comment ID from URL path
	commentID, err := getCommentIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid comment ID")
		return
	}

	// Get existing comment
	comment := models.Comment{}
	if err := comment.GetByID(commentID, userIDPtr); err != nil {
		utils.NotFound(w, "Comment not found")
		return
	}

	// Check ownership
	if comment.UserID != userID {
		utils.Forbidden(w, "You can only delete your own comments")
		return
	}

	// Delete comment
	if err := comment.Delete(); err != nil {
		utils.InternalServerError(w, "Failed to delete comment")
		return
	}

	utils.Success(w, "Comment deleted successfully", nil)
}

// GetCommentsController handles retrieving comments for a post
func GetCommentsController(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	// Get authenticated user
	userID, _ := middleware.GetUserIDFromContext(r)
	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	// Get post ID from URL path or query parameter
	var postID int
	var err error

	// Try to get from URL path first (e.g., /posts/123/comments)
	if strings.Contains(r.URL.Path, "/posts/") {
		postID, err = getPostIDFromCommentsPath(r.URL.Path)
	} else {
		// Get from query parameter
		postIDStr := r.URL.Query().Get("post_id")
		if postIDStr == "" {
			utils.BadRequest(w, "Post ID is required")
			return
		}
		postID, err = strconv.Atoi(postIDStr)
	}

	if err != nil {
		utils.BadRequest(w, "Invalid post ID")
		return
	}

	// Validate post exists
	post := models.Post{}
	if err := post.GetByID(postID, userIDPtr); err != nil {
		utils.NotFound(w, "Post not found")
		return
	}

	// Parse pagination parameters
	query := r.URL.Query()
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

	// Get comments from database
	comments, total, err := models.GetCommentsByPostID(postID, userIDPtr, limit, offset)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve comments")
		return
	}

	// Convert to response format
	var commentResponses []CommentResponse
	for _, comment := range comments {
		commentResponse, err := getCommentResponse(&comment)
		if err != nil {
			utils.InternalServerError(w, "Failed to process comment data")
			return
		}
		commentResponses = append(commentResponses, *commentResponse)
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

	utils.PaginatedSuccess(w, "Comments retrieved successfully", commentResponses, pagination)
}

// VoteCommentController handles comment voting (like/dislike)
func VoteCommentController(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		utils.MethodNotAllowed(w, "Only POST method allowed")
		return
	}

	// Get authenticated user
	userID, _ := middleware.GetUserIDFromContext(r)
	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}
	// Get comment ID from URL path
	commentID, err := getCommentIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid comment ID")
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

	// Check if comment exists
	comment := models.Comment{}
	if err := comment.GetByID(commentID, userIDPtr); err != nil {
		utils.NotFound(w, "Comment not found")
		return
	}

	result, err := models.ToggleCommentVote(userID, commentID, voteData.VoteType)
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

// getCommentIDFromPath extracts comment ID from URL path like /comments/123
func getCommentIDFromPath(path string) (int, error) {
	// Handle paths like /comments/123 or /comments/123/vote
    path = strings.TrimPrefix(path, "/api/comments/")
    parts := strings.Split(path, "/")
    if len(parts) == 0 {
        return 0, errors.New("invalid path format")
    }
    return strconv.Atoi(parts[0])
}

// getPostIDFromCommentsPath extracts post ID from comments URL like /posts/123/comments
func getPostIDFromCommentsPath(path string) (int, error) {
	// Handle paths like /posts/123/comments
	parts := strings.Split(strings.Trim(path, "/"), "/")

	for i, part := range parts {
		if part == "posts" && i+1 < len(parts) {
			return strconv.Atoi(parts[i+1])
		}
	}

	return 0, errors.New("invalid post comments path format")
}

// getCommentResponse converts a Comment model to CommentResponse with additional data
func getCommentResponse(comment *models.Comment) (*CommentResponse, error) {
	// Get author info
	author := models.User{}
	if err := author.GetByID(comment.UserID); err != nil {
		return nil, err
	}

	return &CommentResponse{
		ID:      comment.ID,
		Content: comment.Content,
		PostID:  comment.PostID,
		Author: UserResponse{
			ID:       author.ID,
			Username: author.Username,
			Email:    author.Email,
			JoinedAt: author.CreatedAt,
		},
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}, nil
}
