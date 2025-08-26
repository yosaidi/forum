package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"forum/models"
	"forum/utils"
)

// CategoryResponse represents category data sent to client
type CategoryResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PostCount   int    `json:"post_count"`
}

// GetCategoriesController handles retrieving all categories
func GetCategoriesController(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	// Get categories from database
	categories, err := models.GetAllCategories()
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve categories")
		return
	}

	// Convert to response format
	var categoryResponses []CategoryResponse
	for _, category := range categories {
		categoryResponses = append(categoryResponses, CategoryResponse{
			ID:          category.ID,
			Name:        category.Name,
			Description: category.Description,
			PostCount:   category.PostCount,
		})
	}

	utils.Success(w, "Categories retrieved successfully", categoryResponses)
}

// GetCategoryController handles retrieving a single category
func GetCategoryController(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.MethodNotAllowed(w, "Only GET method allowed")
		return
	}

	// Get category ID from URL path
	categoryID, err := getCategoryIDFromPath(r.URL.Path)
	if err != nil {
		utils.BadRequest(w, "Invalid category ID")
		return
	}

	// Get category from database
	categories, err := models.GetAllCategories()
	if err != nil {
		utils.InternalServerError(w, "Failed to retrieve categories")
		return
	}

	// Find the specific category
	var category *models.Category
	for _, cat := range categories {
		if cat.ID == categoryID {
			category = &cat
			break
		}
	}

	if category == nil {
		utils.NotFound(w, "Category not found")
		return
	}

	categoryResponse := CategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Description: category.Description,
		PostCount:   category.PostCount,
	}

	utils.Success(w, "Category retrieved successfully", categoryResponse)
}

// Helper functions

// getCategoryIDFromPath extracts category ID from URL path like /categories/123
func getCategoryIDFromPath(path string) (int, error) {
	// Remove "/api/categories/" prefix
	path = strings.TrimPrefix(path, "/api/categories/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return 0, errors.New("invalid path format")
	}
	return strconv.Atoi(parts[0])
}
