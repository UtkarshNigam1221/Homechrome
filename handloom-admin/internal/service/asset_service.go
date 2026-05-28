// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/lambdaclient"
	"github.com/handloom/admin/internal/s3client"
	"github.com/handloom/admin/pkg/errors"
)

const (
	presignExpiry   = 15 * time.Minute
	maxImageSize    = 50 << 20 // 50 MB
	maxVideoSize    = 50 << 20 // 50 MB
	maxDocumentSize = 10 << 20 // 10 MB
	tmpPrefix       = "tmp/"
	assetsPrefix    = "assets/"

	contentTypePDF = "application/pdf"

	videoMP4ContentType = "video/mp4"
)

var validDocumentTypes = map[string]bool{
	contentTypePDF:       true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"text/csv": true,
}

// s3Ops abstracts the S3 operations needed by AssetService so it can be mocked in tests.
type s3Ops interface {
	GeneratePresignedPutURL(ctx context.Context, bucket, key, contentType string, expiry time.Duration) (string, error)
	CopyObject(ctx context.Context, bucket, srcKey, dstKey string) error
	DeleteObject(ctx context.Context, bucket, key string) error
}

// resizerInvoker abstracts the Lambda call so it can be mocked in tests.
type resizerInvoker interface {
	InvokeAsync(ctx context.Context, functionName string, payload []byte) error
}

// AssetService handles file uploads via S3.
// Files are uploaded to a tmp/ prefix first, then moved to assets/ on finalize.
// S3 lifecycle auto-deletes tmp/ objects after 24h — no DB scan needed.
type AssetService struct {
	s3Client      s3Ops
	bucket        string
	region        string
	cdnHost       string // CloudFront domain (empty → fall back to direct S3 URL)
	endpoint      string // AWS_ENDPOINT for local dev (empty in production)
	resizer       resizerInvoker
	resizerFnName string // empty disables variant generation (e.g. local dev without LocalStack Lambda)
}

// NewAssetService creates a new AssetService.
func NewAssetService(
	s3Client *s3client.S3Client,
	resizer *lambdaclient.LambdaClient,
	bucket string,
	region string,
	cdnHost string,
	endpoint string,
	resizerFnName string,
) *AssetService {
	return &AssetService{
		s3Client:      s3Client,
		resizer:       resizer,
		bucket:        bucket,
		region:        region,
		cdnHost:       cdnHost,
		endpoint:      strings.TrimRight(endpoint, "/"),
		resizerFnName: resizerFnName,
	}
}

// s3URL builds a public URL for the given key.
// Priority: local endpoint (LocalStack) → CloudFront CDN → direct S3.
func (s *AssetService) s3URL(key string) string {
	if s.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", toBrowserEndpoint(s.endpoint), s.bucket, key)
	}
	if s.cdnHost != "" {
		return fmt.Sprintf("https://%s/%s", s.cdnHost, key)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
}

// trustedURLPrefixes returns the set of URL prefixes that are considered
// trusted asset origins for this service instance. The list mirrors the
// prefixes used by s3URL / keyFromURL so both helpers stay in sync.
func (s *AssetService) trustedURLPrefixes() []string {
	var prefixes []string
	if s.cdnHost != "" {
		prefixes = append(prefixes, fmt.Sprintf("https://%s/", s.cdnHost))
	}
	prefixes = append(prefixes,
		fmt.Sprintf("https://%s.s3.%s.amazonaws.com/", s.bucket, s.region),
		fmt.Sprintf("https://%s.s3.amazonaws.com/", s.bucket),
	)
	if s.endpoint != "" {
		prefixes = append(prefixes, fmt.Sprintf("%s/%s/", toBrowserEndpoint(s.endpoint), s.bucket))
	}
	return prefixes
}

// keyFromURL extracts the S3 object key from a public asset URL.
// Handles CDN URLs, regional S3 URLs, and legacy global S3 URLs.
func (s *AssetService) keyFromURL(assetURL string) string {
	for _, prefix := range s.trustedURLPrefixes() {
		if prefix != "/" && strings.HasPrefix(assetURL, prefix) {
			return strings.TrimPrefix(assetURL, prefix)
		}
	}
	return ""
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
	tmpKey := fmt.Sprintf("%s%s/%s%s", tmpPrefix, req.Type, id, ext)

	// Presigned PUT URL
	uploadURL, err := s.s3Client.GeneratePresignedPutURL(ctx, s.bucket, tmpKey, req.ContentType, presignExpiry)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate presigned URL")
	}

	// In Lambda local mode the SDK uses host.docker.internal which browsers can't reach.
	// Replace with localhost so presigned URLs work in the browser.
	uploadURL = toBrowserURL(uploadURL)

	return &domain.UploadURLResponse{
		UploadURL: uploadURL,
		TmpKey:    tmpKey,
		TmpURL:    s.s3URL(tmpKey),
		ExpiresAt: time.Now().Add(presignExpiry),
	}, nil
}

// FinalizeUpload moves a file from tmp/ to assets/ and returns the public URL.
// Called when a product/category/entity is saved.
func (s *AssetService) FinalizeUpload(ctx context.Context, tmpKey string) (string, error) {
	// Validate the key starts with tmp/
	if !strings.HasPrefix(tmpKey, tmpPrefix) {
		return "", errors.BadRequest("Invalid tmp key")
	}

	// Parse type from key: tmp/{TYPE}/{uuid}.{ext}
	parts := strings.SplitN(tmpKey, "/", 3)
	if len(parts) != 3 {
		return "", errors.BadRequest("Invalid tmp key format")
	}
	assetType := parts[1] // IMAGE, VIDEO, DOCUMENT
	fileName := parts[2]  // uuid.ext

	// Build final key: assets/{TYPE}/{date}/{filename}
	finalKey := fmt.Sprintf("%s%s/%s/%s", assetsPrefix, assetType, time.Now().Format("2006/01/02"), fileName)

	// Copy tmp → assets
	if err := s.s3Client.CopyObject(ctx, s.bucket, tmpKey, finalKey); err != nil {
		return "", errors.Wrap(err, "Failed to copy asset to final location")
	}

	// Image variants are now generated explicitly by callers via SyncImageVariants
	// at save time (product/category update), so we know which images are new vs
	// removed and can skip work on unchanged URLs.

	// Delete tmp object (best-effort; lifecycle will clean up anyway)
	if err := s.s3Client.DeleteObject(ctx, s.bucket, tmpKey); err != nil {
		slog.ErrorContext(ctx, "Failed to delete tmp object", "tmp_key", tmpKey, "error", err)
	}

	finalURL := s.s3URL(finalKey)
	slog.InfoContext(ctx, "Finalized asset", "tmp_key", tmpKey, "final_key", finalKey)
	return finalURL, nil
}

// invokeImageResizer fires the ImageResizer Lambda async (InvocationType=Event)
// with a synthetic S3 event payload, so the caller's API handler isn't blocked
// on the multi-variant resize. Returns once Lambda has queued the request.
func (s *AssetService) invokeImageResizer(ctx context.Context, key string) error {
	if s.resizer == nil || s.resizerFnName == "" {
		// Resizer disabled (local dev without Lambda runtime). Skip silently.
		slog.WarnContext(ctx, "ImageResizer not configured, skipping variant generation", "key", key)
		return nil
	}
	payload := fmt.Sprintf(
		`{"Records":[{"s3":{"bucket":{"name":%q},"object":{"key":%q}}}]}`,
		s.bucket, key,
	)
	start := time.Now()
	err := s.resizer.InvokeAsync(ctx, s.resizerFnName, []byte(payload))
	slog.InfoContext(ctx, "image_resize_dispatch",
		"key", key,
		"duration_ms", time.Since(start).Milliseconds(),
		"error", err,
	)
	return err
}

// FinalizeIfTemp finalizes a tmp/ key into a permanent assets/ URL.
// If value starts with "tmp/", calls FinalizeUpload.
// If value is a trusted asset URL (CDN / S3 / local endpoint), returns as-is.
// Any other non-empty value is passed through with a warning log — hard-rejecting
// would break updates of products that still carry legacy CDN URLs from before a
// CDN_DOMAIN rotation. Handler-level validation is responsible for blocking fresh
// attacker-injected external URLs at the point of upload.
func (s *AssetService) FinalizeIfTemp(ctx context.Context, value string) (string, error) {
	if value == "" {
		return value, nil
	}
	if strings.HasPrefix(value, tmpPrefix) {
		return s.FinalizeUpload(ctx, value)
	}
	for _, prefix := range s.trustedURLPrefixes() {
		if strings.HasPrefix(value, prefix) {
			return value, nil
		}
	}
	// Not tmp/ and not in trusted prefix list. Could be a legacy URL from a
	// previous CDN domain (admin updating an existing product). Log and pass through.
	slog.WarnContext(ctx, "FinalizeIfTemp: untrusted URL pass-through", "url", value)
	return value, nil
}

// imageVariantWidths and imageVariantFormats must match the ImageResizer Lambda
// (lambda/image-resizer/index.mjs). When the resizer changes its output keys,
// update these constants so cleanup stays accurate.
var imageVariantWidths = []int{320, 640, 1080, 1920}

// variantKeysFor returns every S3 key the resizer produces for the given
// original key. Used for cleanup when an image is removed from a product/category.
func variantKeysFor(originalKey string) []string {
	lastDot := strings.LastIndex(originalKey, ".")
	if lastDot == -1 {
		return nil
	}
	stem := originalKey[:lastDot]
	ext := strings.ToLower(originalKey[lastDot+1:])
	rasterFmt := "jpg"
	if ext == "png" {
		rasterFmt = "png"
	}
	formats := []string{"webp", "avif", rasterFmt}
	keys := make([]string, 0, len(imageVariantWidths)*len(formats))
	for _, w := range imageVariantWidths {
		for _, f := range formats {
			keys = append(keys, fmt.Sprintf("%s-%d.%s", stem, w, f))
		}
	}
	return keys
}

// isImageURL returns true if the URL points to an image asset (jpg/png/webp).
func isImageURL(assetURL string) bool {
	lower := strings.ToLower(assetURL)
	// Strip query string before checking extension.
	if idx := strings.Index(lower, "?"); idx != -1 {
		lower = lower[:idx]
	}
	return strings.HasSuffix(lower, ".jpg") ||
		strings.HasSuffix(lower, ".jpeg") ||
		strings.HasSuffix(lower, ".png") ||
		strings.HasSuffix(lower, ".webp")
}

// SyncImageVariants computes the diff between oldURLs and newURLs. For each
// added image URL it invokes the ImageResizer to (re)generate variants. For
// each removed URL it deletes the original asset plus all known variants.
// Best-effort: logs errors and keeps going so partial failures don't strand
// the caller's transaction (DB write already committed by the time we get here).
func (s *AssetService) SyncImageVariants(ctx context.Context, oldURLs, newURLs []string) {
	oldSet := nonEmptyURLSet(oldURLs)
	newSet := nonEmptyURLSet(newURLs)

	for u := range newSet {
		if _, kept := oldSet[u]; kept {
			continue
		}
		s.resizeImageURL(ctx, u)
	}

	for u := range oldSet {
		if _, kept := newSet[u]; kept {
			continue
		}
		s.deleteImageURLWithVariants(ctx, u)
	}
}

// nonEmptyURLSet returns a set of non-empty URLs from the given slice.
func nonEmptyURLSet(urls []string) map[string]struct{} {
	set := make(map[string]struct{}, len(urls))
	for _, u := range urls {
		if u != "" {
			set[u] = struct{}{}
		}
	}
	return set
}

// resizeImageURL invokes the ImageResizer for the given asset URL.
// No-op for non-image URLs or URLs outside the assets/ prefix.
func (s *AssetService) resizeImageURL(ctx context.Context, assetURL string) {
	if !isImageURL(assetURL) {
		return
	}
	key := s.keyFromURL(assetURL)
	if key == "" || !strings.HasPrefix(key, assetsPrefix) {
		return
	}
	if err := s.invokeImageResizer(ctx, key); err != nil {
		slog.WarnContext(ctx, "Failed to resize image", "url", assetURL, "error", err)
	}
}

// deleteImageURLWithVariants removes the original asset plus all known
// resizer-generated variants. Missing variants return success silently
// (S3 DeleteObject is idempotent on non-existent keys).
func (s *AssetService) deleteImageURLWithVariants(ctx context.Context, assetURL string) {
	if !isImageURL(assetURL) {
		return
	}
	key := s.keyFromURL(assetURL)
	if key == "" || !strings.HasPrefix(key, assetsPrefix) {
		return
	}
	if err := s.s3Client.DeleteObject(ctx, s.bucket, key); err != nil {
		slog.WarnContext(ctx, "Failed to delete original image", "key", key, "error", err)
	}
	for _, vk := range variantKeysFor(key) {
		if err := s.s3Client.DeleteObject(ctx, s.bucket, vk); err != nil {
			slog.WarnContext(ctx, "Failed to delete variant", "key", vk, "error", err)
		}
	}
}

// Ensure AssetFinalizer interface compliance
var _ domain.AssetFinalizer = (*AssetService)(nil)

// DeleteAsset deletes a file from the assets/ prefix by its public URL.
// Handles both CDN URLs and direct S3 URLs (for pre-CDN assets).
// For image URLs, also deletes every known resizer-generated variant so we
// never leak orphaned variant objects when callers delete an image directly.
func (s *AssetService) DeleteAsset(ctx context.Context, assetURL string) error {
	key := s.keyFromURL(assetURL)
	if key == "" {
		return errors.BadRequest("Invalid asset URL")
	}

	if !strings.HasPrefix(key, assetsPrefix) {
		return errors.BadRequest("Can only delete files in assets/ prefix")
	}

	if err := s.s3Client.DeleteObject(ctx, s.bucket, key); err != nil {
		return errors.Wrap(err, "Failed to delete asset")
	}

	// Best-effort variant cleanup for image assets.
	if isImageURL(assetURL) {
		for _, vk := range variantKeysFor(key) {
			if err := s.s3Client.DeleteObject(ctx, s.bucket, vk); err != nil {
				slog.WarnContext(ctx, "Failed to delete image variant", "key", vk, "error", err)
			}
		}
	}

	slog.InfoContext(ctx, "Deleted asset", "key", key)
	return nil
}

// Helpers

func isValidContentType(contentType string, assetType domain.AssetType) bool {
	switch assetType {
	case domain.AssetTypeImage:
		return strings.HasPrefix(contentType, "image/")
	case domain.AssetTypeVideo:
		return contentType == videoMP4ContentType
	case domain.AssetTypeDocument:
		return validDocumentTypes[contentType]
	}
	return false
}

func getMaxSize(assetType domain.AssetType) int64 {
	switch assetType {
	case domain.AssetTypeImage:
		return maxImageSize
	case domain.AssetTypeVideo:
		return maxVideoSize
	case domain.AssetTypeDocument:
		return maxDocumentSize
	}
	return maxDocumentSize
}

// toBrowserURL rewrites Docker-internal hostnames in URLs so browsers can reach them.
// e.g. http://host.docker.internal:4566/... → http://localhost:4566/...
func toBrowserURL(u string) string {
	return strings.Replace(u, "host.docker.internal", "localhost", 1)
}

// toBrowserEndpoint rewrites a Docker-internal endpoint for browser use.
func toBrowserEndpoint(endpoint string) string {
	return strings.Replace(endpoint, "host.docker.internal", "localhost", 1)
}
