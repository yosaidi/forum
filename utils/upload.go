package utils

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UploadConfig holds upload configuration
type UploadConfig struct {
	MaxFileSize  int64
	AllowedTypes []string
	UploadDir    string
	URLPrefix    string
}

// Default avatar upload configuration
var AvatarUploadConfig = UploadConfig{
	MaxFileSize:  5 << 20,
	AllowedTypes: []string{"image/jpeg", "image/png", "image/gif", "image/webp"},
	UploadDir:    "./uploads/avatars",
	URLPrefix:    "/uploads/avatars",
}

// UploadResult contains information about uploaded file
type UploadResult struct {
	Filename     string `json:"filename"`
	OriginalName string `json:"original_name"`
	Size         int64  `json:"size"`
	URL          string `json:"url"`
	MimeType     string `json:"mime_type"`
}

// InitUploadDirectories create necessary upload directories
func InitUploadDirectories() error {
	dirs := []string{
		AvatarUploadConfig.UploadDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}
	return nil
}

// HandleFileUpload processes multipart file upload
func HandleFileUpload(r *http.Request, fieldName string, config UploadConfig) (*UploadResult, error) {
	if err := os.MkdirAll(config.UploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %v", err)
	}

	// Parse multipart form with size limit
	err := r.ParseMultipartForm(config.MaxFileSize)
	if err != nil {
		return nil, fmt.Errorf("file too large or invalid form data")
	}

	// Get file from form
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return nil, fmt.Errorf("no file provided or invalid file field")
	}
	defer file.Close()

	// Validate file size
	if header.Size > config.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d MB", config.MaxFileSize/(1<<20))
	}

	// Validate file type
	if !isValidFileType(header, config.AllowedTypes) {
		return nil, fmt.Errorf("invalid file type. Allowed types: %s", strings.Join(config.AllowedTypes, ", "))
	}

	// Generate unique filename
	filename, err := generateUniqueFilename(header.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to generate filename: %v", err)
	}

	// Create full file path
	fullPath := filepath.Join(config.UploadDir, filename)
	
	// Save file to disk
	if err := saveFile(file, fullPath); err != nil {
		return nil, fmt.Errorf("failed to save file: %v", err)
	}

	// Return upload result
	result := &UploadResult{
		Filename:     filename,
		OriginalName: header.Filename,
		Size:         header.Size,
		URL:          config.URLPrefix + "/" + filename,
		MimeType:     header.Header.Get("Content-Type"),
	}

	return result, nil
}

// isValidFileType checks if the uploaded file type is allowed
func isValidFileType(header *multipart.FileHeader, allowedTypes []string) bool {
	// Get content type from header
	contentType := header.Header.Get("Content-Type")

	// If no content type in header, try to detect from filename
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
	}

	// Check against allowed types
	for _, allowed := range allowedTypes {
		if contentType == allowed {
			return true
		}
	}
	return false
}

// generateUniqueFilename creates a unique filename while preserving extension
func generateUniqueFilename(originalName string) (string, error) {
	// Get file extension
	ext := filepath.Ext(originalName)
	if ext == "" {
		return "", fmt.Errorf("file must have an extension")
	}

	// Generate UUID for unique filename
	id := uuid.New()
	timestamp := time.Now().Unix()

	// Create filename: timestamp_uuid.ext
	filename := fmt.Sprintf("%d_%s%s", timestamp, id.String(), ext)

	return filename, nil
}

// saveFile saves uploaded file to specified path
func saveFile(src multipart.File, dst string) error {
	// Create destination file
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy file content
	_, err = io.Copy(out, src)
	return err
}

// DeleteFile removes a file from the filesystem
func DeleteFile(filepath string) error {
	// Security check: ensure file is in allowed directory
	if !strings.HasPrefix(filepath, "./uploads/") {
		return fmt.Errorf("file path not in allowed directory")
	}

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil // File doesn't exist, consider it deleted
	}

	return os.Remove(filepath)
}

// GetFileInfo returns information about an uploaded file
func GetFileInfo(filename string, config UploadConfig) (*UploadResult, error) {
	filePath := filepath.Join(config.UploadDir, filename)

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	result := &UploadResult{
		Filename: filename,
		Size:     info.Size(),
		URL:      config.URLPrefix + "/" + filename,
		MimeType: mime.TypeByExtension(filepath.Ext(filename)),
	}

	return result, nil
}

// CleanupOldFiles removes files older than specified duration
func CleanupOldFiles(directory string, maxAge time.Duration) error {
	return filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is older than maxAge
		if time.Since(info.ModTime()) > maxAge {
			if err := os.Remove(path); err != nil {
				// Log error but continue cleanup
				fmt.Printf("Warning: failed to remove old file %s: %v\n", path, err)
			}
		}

		return nil
	})
}

// ValidateImageFile validates that uploaded file is a valid image
func ValidateImageFile(file multipart.File) error {
	// Read first few bytes to check file signature
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return err
	}

	// Reset file pointer
	file.Seek(0, 0)

	// Detect content type
	contentType := http.DetectContentType(buffer)

	// Check if it's an image
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("file is not a valid image")
	}

	return nil
}

// GetAvatarFilePath returns the full file path for an avatar
func GetAvatarFilePath(filename string) string {
	return filepath.Join(AvatarUploadConfig.UploadDir, filename)
}

// ExtractFilenameFromURL extracts filename from avatar URL
func ExtractFilenameFromURL(url string) string {
	if url == "" {
		return ""
	}

	// Remove URL prefix to get filename
	filename := strings.TrimPrefix(url, AvatarUploadConfig.URLPrefix+"/")
	filename = strings.TrimPrefix(filename, "/")

	return filename
}
