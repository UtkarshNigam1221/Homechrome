// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/s3client"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// AssetService handles file uploads via S3.
// Files are uploaded to a tmp/ prefix first, then moved to assets/ on finalize.
// S3 lifecycle auto-deletes tmp/ objects after 24h — no DB scan needed.
type AssetService struct {
	s3Client *s3client.S3Client
	logger   *logger.Logger
	bucket   string
}

// NewAssetService creates a new AssetService
func NewAssetService(
	logger *logger.Logger,
	s3Client *s3client.S3Client,
	bucket string,
) *AssetService {
	return &AssetService{
		s3Client: s3Client,
		logger:   logger,
		bucket:   bucket,
	}
}

// GetUploadURL generates a presigned PUT URL for a temporary upload.
// The file is placed under tmp/{TYPE}/{uuid}.{ext}.
func (s *AssetService) GetUploadURL(ctx context.Context, req domain.UploadAssetRequest) (*domain.UploadURLResponse, error) {
	if !isValidContentType(req.ContentType, req.Type) {
		return nil, errors.BadRequest("Invalid content type for asset type")
	}

	maxSize := getMaxSize(req.Type)
	if req.Size > maxSize {
		return nil, errors.BadRequest(fmt.Sprintf("File size exceeds maximum allowed (%d bytes)", maxSize))
	}

	// Generate a unique key under tmp/
	id := uuid.New().String()
	ext := filepath.Ext(req.FileName)
	tmpKey := fmt.Sprintf("tmp/%s/%s%s", req.Type, id, ext)

	// Presigned PUT URL (15 min expiry)
	uploadURL, err := s.s3Client.GeneratePresignedPutURL(ctx, s.bucket, tmpKey, req.ContentType, 15*time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate presigned URL")
	}

	// Temporary URL for preview (public-read won't work on tmp/, but frontend can use the local blob)
	tmpURL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, tmpKey)

	return &domain.UploadURLResponse{
		UploadURL: uploadURL,
		TmpKey:    tmpKey,
		TmpURL:    tmpURL,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}, nil
}

// FinalizeUpload moves a file from tmp/ to assets/ and returns the public URL.
// Called when a product/category/entity is saved.
func (s *AssetService) FinalizeUpload(ctx context.Context, tmpKey string) (string, error) {
	// Validate the key starts with tmp/
	if !strings.HasPrefix(tmpKey, "tmp/") {
		return "", errors.BadRequest("Invalid tmp key")
	}

	// Parse type from key: tmp/{TYPE}/{uuid}.{ext}
	parts := strings.SplitN(tmpKey, "/", 3)
	if len(parts) != 3 {
		return "", errors.BadRequest("Invalid tmp key format")
	}
	assetType := parts[1]  // IMAGE, VIDEO, DOCUMENT
	fileName := parts[2]   // uuid.ext

	// Build final key: assets/{TYPE}/{date}/{filename}
	finalKey := fmt.Sprintf("assets/%s/%s/%s", assetType, time.Now().Format("2006/01/02"), fileName)

	// Copy tmp → assets
	if err := s.s3Client.CopyObject(ctx, s.bucket, tmpKey, finalKey); err != nil {
		return "", errors.Wrap(err, "Failed to copy asset to final location")
	}

	// Delete tmp object (best-effort; lifecycle will clean up anyway)
	if err := s.s3Client.DeleteObject(ctx, s.bucket, tmpKey); err != nil {
		s.logger.WithContext(ctx).Errorf("Failed to delete tmp object %s: %v", tmpKey, err)
	}

	finalURL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, finalKey)
	s.logger.WithContext(ctx).Infof("Finalized asset: %s → %s", tmpKey, finalKey)
	return finalURL, nil
}

// FinalizeIfTemp finalizes a tmp/ key into a permanent assets/ URL.
// If value starts with "tmp/", calls FinalizeUpload. Otherwise returns as-is.
func (s *AssetService) FinalizeIfTemp(ctx context.Context, value string) (string, error) {
	if strings.HasPrefix(value, "tmp/") {
		return s.FinalizeUpload(ctx, value)
	}
	return value, nil
}

// Ensure AssetFinalizer interface compliance
var _ domain.AssetFinalizer = (*AssetService)(nil)

// DeleteAsset deletes a file from the assets/ prefix by its public URL.
func (s *AssetService) DeleteAsset(ctx context.Context, assetURL string) error {
	// Parse key from URL: https://{bucket}.s3.amazonaws.com/{key}
	prefix := fmt.Sprintf("https://%s.s3.amazonaws.com/", s.bucket)
	if !strings.HasPrefix(assetURL, prefix) {
		return errors.BadRequest("Invalid asset URL")
	}

	key := strings.TrimPrefix(assetURL, prefix)
	if !strings.HasPrefix(key, "assets/") {
		return errors.BadRequest("Can only delete files in assets/ prefix")
	}

	if err := s.s3Client.DeleteObject(ctx, s.bucket, key); err != nil {
		return errors.Wrap(err, "Failed to delete asset")
	}

	s.logger.WithContext(ctx).Infof("Deleted asset: %s", key)
	return nil
}

// Helpers

func isValidContentType(contentType string, assetType domain.AssetType) bool {
	switch assetType {
	case domain.AssetTypeImage:
		return strings.HasPrefix(contentType, "image/")
	case domain.AssetTypeVideo:
		return strings.HasPrefix(contentType, "video/")
	case domain.AssetTypeDocument:
		validTypes := []string{
			"application/pdf",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"text/csv",
		}
		for _, t := range validTypes {
			if contentType == t {
				return true
			}
		}
		return false
	}
	return false
}

func getMaxSize(assetType domain.AssetType) int64 {
	switch assetType {
	case domain.AssetTypeImage:
		return 50 * 1024 * 1024 // 50MB
	case domain.AssetTypeVideo:
		return 100 * 1024 * 1024 // 100MB
	case domain.AssetTypeDocument:
		return 10 * 1024 * 1024 // 10MB
	}
	return 10 * 1024 * 1024
}
