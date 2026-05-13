// Package handler provides HTTP handlers
package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/handloom/admin/pkg/response"
)

// maxUploadBytes caps both MaxBytesReader and ParseMultipartForm so they
// stay in lockstep — drifting these is what trips gosec G120.
const (
	maxUploadBytes = 50 << 20 // 50 MB
	contentTypePNG = "image/png"
)

// UploadHandler handles file upload requests
type UploadHandler struct {
	uploadDir string
	baseURL   string
}

// NewUploadHandler creates a new UploadHandler
func NewUploadHandler(uploadDir, baseURL string) *UploadHandler {
	// Ensure upload directory exists
	if err := os.MkdirAll(uploadDir, 0750); err != nil {
		panic(fmt.Sprintf("Failed to create upload directory: %v", err))
	}
	return &UploadHandler{
		uploadDir: uploadDir,
		baseURL:   baseURL,
	}
}

// UploadResponse represents the response after a successful upload
type UploadResponse struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

// Upload handles file uploads
// POST /uploads
func (h *UploadHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	// MaxBytesReader above caps the body so gosec's G120 (unbounded multipart
	// parse) is not exploitable here.
	//nolint:gosec // G120: body is already bounded via MaxBytesReader.
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		response.Error(w, fmt.Errorf("failed to parse form: %w", err))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, fmt.Errorf("failed to get file: %w", err))
		return
	}
	defer func() { _ = file.Close() }()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !h.isAllowedContentType(contentType) {
		response.Error(w, fmt.Errorf("file type not allowed: %s", contentType))
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = h.getExtFromMimeType(contentType)
	}
	filename := fmt.Sprintf("%s_%s%s", time.Now().Format("20060102"), uuid.New().String()[:8], ext)

	// Create subdirectory based on date
	subDir := time.Now().Format("2006/01")
	fullDir := filepath.Join(h.uploadDir, subDir)
	if mkdirErr := os.MkdirAll(fullDir, 0750); mkdirErr != nil {
		response.Error(w, fmt.Errorf("failed to create directory: %w", mkdirErr))
		return
	}

	// Create destination file
	dstPath := filepath.Clean(filepath.Join(fullDir, filename))
	dst, err := os.Create(dstPath)
	if err != nil {
		response.Error(w, fmt.Errorf("failed to create file: %w", err))
		return
	}
	defer func() { _ = dst.Close() }()

	// Copy file content
	size, err := io.Copy(dst, file)
	if err != nil {
		_ = os.Remove(dstPath)
		response.Error(w, fmt.Errorf("failed to save file: %w", err))
		return
	}

	// Build response URL
	relPath := filepath.Join(subDir, filename)
	url := fmt.Sprintf("%s/uploads/%s", h.baseURL, strings.ReplaceAll(relPath, "\\", "/"))

	resp := UploadResponse{
		URL:      url,
		Filename: filename,
		Size:     size,
		MimeType: contentType,
	}

	response.JSON(w, http.StatusOK, resp)
}

// ServeFile serves uploaded files
// GET /uploads/{path}
func (h *UploadHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	// Get the file path from URL
	filePath := chi.URLParam(r, "*")
	if filePath == "" {
		http.NotFound(w, r)
		return
	}

	// Clean the path to prevent directory traversal
	filePath = filepath.Clean(filePath)
	fullPath := filepath.Join(h.uploadDir, filePath)

	// Ensure the path is within upload directory
	if !strings.HasPrefix(fullPath, filepath.Clean(h.uploadDir)) {
		http.NotFound(w, r)
		return
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Set cache headers for images
	w.Header().Set("Cache-Control", "public, max-age=31536000")

	http.ServeFile(w, r, fullPath)
}

// Routes returns the upload routes
func (h *UploadHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.Upload)
	r.Post("/base64", h.UploadFromBase64)
	r.Get("/*", h.ServeFile)
	return r
}

func (h *UploadHandler) isAllowedContentType(contentType string) bool {
	allowed := []string{
		"image/jpeg",
		contentTypePNG,
		"image/gif",
		"image/webp",
		"image/svg+xml",
	}
	for _, a := range allowed {
		if contentType == a {
			return true
		}
	}
	return false
}

func (h *UploadHandler) getExtFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case contentTypePNG:
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	default:
		return ""
	}
}

// UploadFromBase64 handles base64 encoded image uploads
// POST /uploads/base64
func (h *UploadHandler) UploadFromBase64(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data     string `json:"data"`     // base64 encoded data or data URL
		Filename string `json:"filename"` // optional original filename
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, fmt.Errorf("invalid request body: %w", err))
		return
	}

	if req.Data == "" {
		response.Error(w, fmt.Errorf("data is required"))
		return
	}

	// Parse data URL format: data:image/png;base64,xxxxx
	var mimeType string
	var base64Data string

	if strings.HasPrefix(req.Data, "data:") {
		parts := strings.SplitN(req.Data, ",", 2)
		if len(parts) != 2 {
			response.Error(w, fmt.Errorf("invalid data URL format"))
			return
		}
		// Extract mime type from "data:image/png;base64"
		header := parts[0]
		if idx := strings.Index(header, ";"); idx > 0 {
			mimeType = header[5:idx] // Skip "data:"
		}
		base64Data = parts[1]
	} else {
		base64Data = req.Data
		mimeType = contentTypePNG // Default
	}

	// Validate mime type
	if !h.isAllowedContentType(mimeType) {
		response.Error(w, fmt.Errorf("file type not allowed: %s", mimeType))
		return
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		response.Error(w, fmt.Errorf("failed to decode base64: %w", err))
		return
	}

	// Generate filename
	ext := h.getExtFromMimeType(mimeType)
	filename := fmt.Sprintf("%s_%s%s", time.Now().Format("20060102"), uuid.New().String()[:8], ext)

	// Create subdirectory
	subDir := time.Now().Format("2006/01")
	fullDir := filepath.Join(h.uploadDir, subDir)
	if err := os.MkdirAll(fullDir, 0750); err != nil {
		response.Error(w, fmt.Errorf("failed to create directory: %w", err))
		return
	}

	// Write file
	dstPath := filepath.Join(fullDir, filename)
	if err := os.WriteFile(dstPath, data, 0600); err != nil {
		response.Error(w, fmt.Errorf("failed to save file: %w", err))
		return
	}

	// Build response URL
	relPath := filepath.Join(subDir, filename)
	url := fmt.Sprintf("%s/uploads/%s", h.baseURL, strings.ReplaceAll(relPath, "\\", "/"))

	resp := UploadResponse{
		URL:      url,
		Filename: filename,
		Size:     int64(len(data)),
		MimeType: mimeType,
	}

	response.JSON(w, http.StatusOK, resp)
}
