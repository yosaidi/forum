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
	Title       string `json:"title"`
	Content     string `json:"content"`
	CategoryIDs []int  `json:"category_ids"`
}

// PostUpdateRequest represents the JSON structure for updating posts
type PostUpdateRequest struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	CategoryIDs []int  `json:"category_ids"`
}

// PostResponse represents post data sent to client
type PostResponse struct {
	ID           int             `json:"id"`
	Title        string          `json:"title"`
	Content      string          `json:"content"`
	Categories   []CategoryBrief `json:"categories"`
	Author       UserResponse    `json:"author"`
	LikeCount    int             `json:"like_count"`
	DislikeCount int             `json:"dislike_count"`
	CommentCount int             `json:"comment_count"`
	UserVote     *string         `json:"user_vote"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// CategoryBrief for embedding in post responses
type CategoryBrief struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// CreatePostController handles post creation
func CreatePostController(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.MethodNotAllowed(w, "Only POST method allowed")
		return
	}

	userID, exists := middleware.GetUserIDFromContext(r)
	if !exists {
		utils.Unauthorized(w, "Authentication required")
		return
	}

	var req PostCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	// Validate post form (you'll need to update ValidatePostForm)
	if errors := utils.ValidatePostForm(req.Title, req.Content, req.CategoryIDs); errors.HasErrors() {
		utils.ValidationError(w, errors)
		return
	}

	// Create new post
	post := models.Post{
		Title:      req.Title,
		Content:    req.Content,
		UserID:     userID,
		Categories: make([]models.Category, 0, len(req.CategoryIDs)),
	}

	// Convert category IDs to Category objects
	for _, catID := range req.CategoryIDs {
		post.Categories = append(post.Categories, models.Category{
			ID: catID,
		})
	}

	if err := post.Create(); err != nil {
		utils.InternalServerError(w, "Failed to create post")
		return
	}

	// Get full post details for response
	postResponse, err := getPostResponse(&post, userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve post details")
		return
	}
	utils.Created(w, "Post created successfully", postResponse)
}

// GetPostController handles retrieving a single post
func GetPostController(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	postID, err := getPostIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid post ID")
		return
	}

	userID, _ := middleware.GetUserIDFromContext(r)
	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	post := models.Post{}
	if err := post.GetByID(postID, userIDPtr); err != nil {
		utils.NotFound(w, "Post not found")
		return
	}

	postResponse, err := getPostResponse(&post, userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve post details")
		return
	}

	utils.Success(w, "Post retrieved successfully", postResponse)
}

// UpdatePostController handles post updates
func UpdatePostController(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.MethodNotAllowed(w, "Only PUT method allowed")
		return
	}

	userID, exists := middleware.GetUserIDFromContext(r)
	if !exists {
		utils.Unauthorized(w, "Authentication required")
		return
	}

	postID, err := getPostIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid post ID")
		return
	}

	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	post := models.Post{}
	if err := post.GetByID(postID, userIDPtr); err != nil {
		utils.NotFound(w, "Post not found")
		return
	}

	if post.UserID != userID {
		utils.Forbidden(w, "You can only edit your own posts")
		return
	}

	var req PostUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	// Validate post form (you'll need to update ValidatePostForm)
	if errors := utils.ValidatePostForm(req.Title, req.Content, req.CategoryIDs); errors.HasErrors() {
		utils.ValidationError(w, errors)
		return
	}

	// Update post fields
	post.Title = req.Title
	post.Content = req.Content
	post.Categories = make([]models.Category, 0, len(req.CategoryIDs))

	for _, catID := range req.CategoryIDs {
		post.Categories = append(post.Categories, models.Category{ID: catID})
	}

	if err := post.Update(); err != nil {
		utils.InternalServerError(w, "Failed to update post")
		return
	}

	postResponse, err := getPostResponse(&post, userID)
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve updated post details")
		return
	}

	utils.Success(w, "Post updated successfully", postResponse)
}

// DeletePostController handles post deletion
func DeletePostController(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.MethodNotAllowed(w, "Only DELETE method allowed")
		return
	}

	userID, exists := middleware.GetUserIDFromContext(r)
	if !exists {
		utils.Unauthorized(w, "Authentication required")
		return
	}

	postID, err := getPostIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid post ID")
		return
	}

	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	post := models.Post{}
	if err := post.GetByID(postID, userIDPtr); err != nil {
		utils.NotFound(w, "Post not found")
		return
	}

	if post.UserID != userID {
		utils.Forbidden(w, "You can only delete your own posts")
		return
	}

	if err := post.Delete(); err != nil {
		utils.InternalServerError(w, "Failed to delete post")
		return
	}

	utils.Success(w, "Post deleted successfully", nil)
}

// GetPostsController handles retrieving multiple posts with filtering
func GetPostsController(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	query := r.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	if page <= 0 {
		page = 1
	}

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	limit, offset, err := utils.ValidatePagination(page, limit)
	if err != nil {
		utils.BadRequest(w, err.Error())
		return
	}

	categoryID, _ := strconv.Atoi(query.Get("category"))
	authorID, _ := strconv.Atoi(query.Get("author"))
	sortBy := query.Get("sort")

	if sortBy == "" {
		sortBy = "newest"
	}

	userID, _ := middleware.GetUserIDFromContext(r)

	posts, total, err := models.GetPosts(models.PostFilters{
		CurrentUserID: userID,
		CategoryID:    categoryID,
		AuthorID:      authorID,
		SortBy:        sortBy,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve posts")
		return
	}

	var postResponses []PostResponse
	for _, post := range posts {
		postResponse, err := getPostResponse(&post, userID)
		if err != nil {
			utils.InternalServerError(w, "Failed to process post data")
			return
		}
		postResponses = append(postResponses, *postResponse)
	}

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
	if r.Method != http.MethodPost {
		utils.MethodNotAllowed(w, "Only POST method allowed")
		return
	}

	userID, exists := middleware.GetUserIDFromContext(r)
	if !exists {
		utils.Unauthorized(w, "Authentication required")
		return
	}

	postID, err := getPostIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid post ID")
		return
	}

	var voteData struct {
		VoteType string `json:"vote_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&voteData); err != nil {
		utils.BadRequest(w, "Invalid JSON format")
		return
	}

	if !utils.IsValidVoteType(voteData.VoteType) {
		utils.BadRequest(w, "Vote type must be 'like' or 'dislike'")
		return
	}

	var userIDPtr *int
	if userID > 0 {
		userIDPtr = &userID
	}

	post := models.Post{}
	if err := post.GetByID(postID, userIDPtr); err != nil {
		utils.NotFound(w, "Post not found")
		return
	}

	result, err := models.TogglePostVote(userID, postID, voteData.VoteType)
	if err != nil {
		utils.InternalServerError(w, "Failed to process vote")
		return
	}

	utils.Success(w, "Vote processed successfully", map[string]interface{}{
		"action":        result.Action,
		"vote_type":     result.VoteType,
		"like_count":    result.NewLikes,
		"dislike_count": result.NewDislikes,
	})
}

// Helper functions

func getPostIDFromPath(path string) (int, error) {
	path = strings.TrimPrefix(path, "/api/posts/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return 0, errors.New("invalid path format")
	}
	return strconv.Atoi(parts[0])
}

func getPostResponse(post *models.Post, currentUserID int) (*PostResponse, error) {
	author := models.User{}
	if err := author.GetByID(post.UserID); err != nil {
		return nil, err
	}

	likeCount, dislikeCount, err := post.GetVoteCounts()
	if err != nil {
		return nil, err
	}

	commentCount, err := post.GetCommentCount()
	if err != nil {
		return nil, err
	}

	var userVote *string
	if currentUserID > 0 {
		vote := models.Vote{}
		if err := vote.GetByUserAndPost(currentUserID, post.ID); err == nil {
			userVote = &vote.VoteType
		}
	}

	// Map categories from post
	categories := make([]CategoryBrief, 0, len(post.Categories))
	for _, cat := range post.Categories {
		categories = append(categories, CategoryBrief{
			ID:   cat.ID,
			Name: cat.Name,
		})
	}

	return &PostResponse{
		ID:         post.ID,
		Title:      post.Title,
		Content:    post.Content,
		Categories: categories, // Changed from single category
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
