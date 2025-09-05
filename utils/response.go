package utils

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Success sends a successful JSON response
func Success(w http.ResponseWriter, message string, data interface{}) {
	response := APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	sendJSON(w, http.StatusOK, response)
}

// Created sends a 201 Created JSON response
func Created(w http.ResponseWriter, message string, data interface{}) {
	response := APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	sendJSON(w, http.StatusCreated, response)
}

// Error sends an error JSON response with custom status code
func Error(w http.ResponseWriter, statusCode int, message string) {
	response := APIResponse{
		Success: false,
		Message: "Request failed",
		Error:   message,
	}
	sendJSON(w, statusCode, response)
}

// BadRequest sends a 400 Bad Request JSON response
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, message)
}

// Unauthorized sends a 401 Unauthorized JSON response
func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, message)
}

// Forbidden sends a 403 Forbidden JSON response
func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, message)
}

// NotFound sends a 404 Not Found JSON response
func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, message)
}

// Conflict sends a 409 Conflict JSON response
func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, message)
}

// InternalServerError sends a 500 Internal Server Error JSON response
func InternalServerError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, message)
}

// MethodNotAllowed send a 405 Method Not Allowed JSON response
func MethodNotAllowed(w http.ResponseWriter, message string) {
	Error(w, http.StatusMethodNotAllowed, message)
}

// ValidationError sends a 422 Unprocessable Entity JSON response
// Used for validation errors with detailed field information
func ValidationError(w http.ResponseWriter, errors map[string]string) {
	response := APIResponse{
		Success: false,
		Message: "Validation failed",
		Data:    errors,
	}

	sendJSON(w, http.StatusUnprocessableEntity, response)
}

// PaginatedSuccess sends a successful JSON response with pagination info
func PaginatedSuccess(w http.ResponseWriter, message string, data interface{}, pagination interface{}) {
	response := struct {
		Success    bool        `json:"success"`
		Message    string      `json:"message"`
		Data       interface{} `json:"data"`
		Pagination interface{} `json:"pagination"`
	}{
		Success:    true,
		Message:    message,
		Data:       data,
		Pagination: pagination,
	}

	sendJSON(w, http.StatusOK, response)
}

// sendJSON is a helper function that sends JSON responses
func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"message":"Internal server error","error":"Failed to encode response"}`))
	}
}

// ParseJSON is a helper function to parse JSON request bodies
func ParseJSON(r *http.Request, v interface{}) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return errors.New("content-Type must be application/json")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		return err
	}

	return nil
}

// Add to utils or create route_helpers.go
func GetIDFromURL(r *http.Request, prefix string) (int, error) {
	path := strings.TrimPrefix(r.URL.Path, "/api")
	remaining := strings.TrimPrefix(path, prefix)
	parts := strings.Split(remaining, "/")
	if len(parts) == 0 {
		return 0, errors.New("no ID found")
	}
	return strconv.Atoi(parts[0])
}
