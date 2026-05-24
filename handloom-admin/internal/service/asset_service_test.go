package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// ---------------------------------------------------------------------------
// Mock
// ---------------------------------------------------------------------------

type mockS3Client struct {
	mock.Mock
}

func (m *mockS3Client) GeneratePresignedPutURL(ctx context.Context, bucket, key, contentType string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, bucket, key, contentType, expiry)
	return args.String(0), args.Error(1)
}

func (m *mockS3Client) CopyObject(ctx context.Context, bucket, srcKey, dstKey string) error {
	args := m.Called(ctx, bucket, srcKey, dstKey)
	return args.Error(0)
}

func (m *mockS3Client) DeleteObject(ctx context.Context, bucket, key string) error {
	args := m.Called(ctx, bucket, key)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// mockResizer
// ---------------------------------------------------------------------------

type resizerCall struct {
	FunctionName string
	Payload      []byte
}

type mockResizer struct {
	invocations []resizerCall
	returnErr   error
}

func (m *mockResizer) InvokeSync(_ context.Context, fn string, payload []byte) ([]byte, error) {
	m.invocations = append(m.invocations, resizerCall{FunctionName: fn, Payload: payload})
	if m.returnErr != nil {
		return nil, m.returnErr
	}
	return []byte(`{}`), nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const testBucket = "test-bucket"
const testRegion = "ap-south-1"

func newTestAssetService(s3 *mockS3Client, resizer resizerInvoker) *AssetService {
	return &AssetService{
		s3Client:      s3,
		resizer:       resizer,
		bucket:        testBucket,
		region:        testRegion,
		resizerFnName: "test-resizer-fn",
	}
}

func s3URL(key string) string {
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", testBucket, testRegion, key)
}

// ---------------------------------------------------------------------------
// TestAssetService_GetUploadURL
// ---------------------------------------------------------------------------

func TestAssetService_GetUploadURL(t *testing.T) {
	tests := []struct {
		name        string
		req         domain.UploadAssetRequest
		presignURL  string
		presignErr  error
		wantErr     bool
		errContains string
	}{
		{
			name: "valid image upload",
			req: domain.UploadAssetRequest{
				FileName:    "photo.jpg",
				ContentType: "image/jpeg",
				Size:        1024,
				Type:        domain.AssetTypeImage,
			},
			presignURL: "https://s3.presigned.url/put",
		},
		{
			name: "valid video upload",
			req: domain.UploadAssetRequest{
				FileName:    "clip.mp4",
				ContentType: "video/mp4",
				Size:        50 << 20,
				Type:        domain.AssetTypeVideo,
			},
			presignURL: "https://s3.presigned.url/put",
		},
		{
			name: "valid document upload (pdf)",
			req: domain.UploadAssetRequest{
				FileName:    "report.pdf",
				ContentType: contentTypePDF,
				Size:        5 << 20,
				Type:        domain.AssetTypeDocument,
			},
			presignURL: "https://s3.presigned.url/put",
		},
		{
			name: "invalid content type for image (video/mp4 as IMAGE)",
			req: domain.UploadAssetRequest{
				FileName:    "clip.mp4",
				ContentType: "video/mp4",
				Size:        1024,
				Type:        domain.AssetTypeImage,
			},
			wantErr:     true,
			errContains: "Invalid content type",
		},
		{
			name: "file too large - image > 50MB",
			req: domain.UploadAssetRequest{
				FileName:    "huge.png",
				ContentType: "image/png",
				Size:        maxImageSize + 1,
				Type:        domain.AssetTypeImage,
			},
			wantErr:     true,
			errContains: "File size exceeds maximum",
		},
		{
			name: "file too large - video > 50MB",
			req: domain.UploadAssetRequest{
				FileName:    "huge.mp4",
				ContentType: "video/mp4",
				Size:        maxVideoSize + 1,
				Type:        domain.AssetTypeVideo,
			},
			wantErr:     true,
			errContains: "File size exceeds maximum",
		},
		{
			name: "file too large - document > 10MB",
			req: domain.UploadAssetRequest{
				FileName:    "huge.pdf",
				ContentType: contentTypePDF,
				Size:        maxDocumentSize + 1,
				Type:        domain.AssetTypeDocument,
			},
			wantErr:     true,
			errContains: "File size exceeds maximum",
		},
		{
			name: "presign error from S3",
			req: domain.UploadAssetRequest{
				FileName:    "photo.jpg",
				ContentType: "image/jpeg",
				Size:        1024,
				Type:        domain.AssetTypeImage,
			},
			presignErr:  fmt.Errorf("s3 unavailable"),
			wantErr:     true,
			errContains: "Failed to generate presigned URL",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s3Mock := new(mockS3Client)
			svc := newTestAssetService(s3Mock, nil)
			ctx := context.Background()

			// Only set up presign expectation when we expect the call to reach S3
			if !tc.wantErr || tc.presignErr != nil {
				s3Mock.On("GeneratePresignedPutURL",
					mock.Anything, testBucket, mock.AnythingOfType("string"),
					tc.req.ContentType, mock.AnythingOfType("time.Duration"),
				).Return(tc.presignURL, tc.presignErr)
			}

			resp, err := svc.GetUploadURL(ctx, tc.req)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)

			assert.Equal(t, tc.presignURL, resp.UploadURL)
			assert.True(t, strings.HasPrefix(resp.TmpKey, tmpPrefix+string(tc.req.Type)+"/"))
			assert.Equal(t, s3URL(resp.TmpKey), resp.TmpURL)
			assert.WithinDuration(t, time.Now().Add(presignExpiry), resp.ExpiresAt, 5*time.Second)

			s3Mock.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAssetService_FinalizeUpload
// ---------------------------------------------------------------------------

func TestAssetService_FinalizeUpload(t *testing.T) {
	tests := []struct {
		name        string
		tmpKey      string
		copyErr     error
		deleteErr   error
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful finalize",
			tmpKey: "tmp/IMAGE/abc-123.jpg",
		},
		{
			name:        "key not starting with tmp/",
			tmpKey:      "assets/IMAGE/abc-123.jpg",
			wantErr:     true,
			errContains: "Invalid tmp key",
		},
		{
			name:        "invalid tmp key format - wrong parts count",
			tmpKey:      "tmp/nofilename",
			wantErr:     true,
			errContains: "Invalid tmp key format",
		},
		{
			name:        "s3 copy error",
			tmpKey:      "tmp/IMAGE/abc-123.jpg",
			copyErr:     fmt.Errorf("copy failed"),
			wantErr:     true,
			errContains: "Failed to copy asset",
		},
		{
			name:      "s3 delete error still succeeds",
			tmpKey:    "tmp/IMAGE/abc-123.jpg",
			deleteErr: fmt.Errorf("delete failed"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s3Mock := new(mockS3Client)
			svc := newTestAssetService(s3Mock, nil)
			ctx := context.Background()

			// Only set up S3 expectations for cases that reach S3 calls
			if tc.tmpKey == "tmp/IMAGE/abc-123.jpg" || strings.HasPrefix(tc.tmpKey, "tmp/") && strings.Count(tc.tmpKey, "/") >= 2 {
				// Parse expected final key parts
				parts := strings.SplitN(tc.tmpKey, "/", 3)
				if len(parts) == 3 {
					expectedFinalPrefix := fmt.Sprintf("assets/%s/%s/", parts[1], time.Now().Format("2006/01/02"))

					s3Mock.On("CopyObject",
						mock.Anything, testBucket, tc.tmpKey,
						mock.MatchedBy(func(dstKey string) bool {
							return strings.HasPrefix(dstKey, expectedFinalPrefix)
						}),
					).Return(tc.copyErr)

					if tc.copyErr == nil {
						s3Mock.On("DeleteObject",
							mock.Anything, testBucket, tc.tmpKey,
						).Return(tc.deleteErr)
					}
				}
			}

			finalURL, err := svc.FinalizeUpload(ctx, tc.tmpKey)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				assert.Empty(t, finalURL)
				return
			}

			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(finalURL, s3URL("assets/")),
				"final URL should start with assets/ prefix, got: %s", finalURL)
			assert.Contains(t, finalURL, time.Now().Format("2006/01/02"))

			s3Mock.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAssetService_FinalizeIfTemp
// ---------------------------------------------------------------------------

func TestAssetService_FinalizeIfTemp(t *testing.T) {
	t.Run("tmp/ prefixed key delegates to FinalizeUpload", func(t *testing.T) {
		s3Mock := new(mockS3Client)
		svc := newTestAssetService(s3Mock, nil)
		ctx := context.Background()

		tmpKey := "tmp/IMAGE/abc-123.jpg"

		s3Mock.On("CopyObject",
			mock.Anything, testBucket, tmpKey, mock.AnythingOfType("string"),
		).Return(nil)
		s3Mock.On("DeleteObject",
			mock.Anything, testBucket, tmpKey,
		).Return(nil)

		result, err := svc.FinalizeIfTemp(ctx, tmpKey)

		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(result, s3URL("assets/")))
		s3Mock.AssertExpectations(t)
	})

	t.Run("non-tmp value returned as-is", func(t *testing.T) {
		s3Mock := new(mockS3Client)
		svc := newTestAssetService(s3Mock, nil)
		ctx := context.Background()

		value := "https://test-bucket.s3.amazonaws.com/assets/IMAGE/2024/01/01/photo.jpg"
		result, err := svc.FinalizeIfTemp(ctx, value)

		require.NoError(t, err)
		assert.Equal(t, value, result)
		// No S3 calls should have been made
		s3Mock.AssertNotCalled(t, "CopyObject")
		s3Mock.AssertNotCalled(t, "DeleteObject")
	})
}

// ---------------------------------------------------------------------------
// TestAssetService_FinalizeIfTemp_URLAllowlist
// ---------------------------------------------------------------------------

func TestAssetService_FinalizeIfTemp_URLAllowlist(t *testing.T) {
	t.Run("empty string passes through unchanged", func(t *testing.T) {
		s3Mock := new(mockS3Client)
		svc := newTestAssetService(s3Mock, nil)

		result, err := svc.FinalizeIfTemp(context.Background(), "")

		require.NoError(t, err)
		assert.Equal(t, "", result)
		s3Mock.AssertNotCalled(t, "CopyObject")
	})

	t.Run("tmp key delegates to FinalizeUpload", func(t *testing.T) {
		s3Mock := new(mockS3Client)
		svc := newTestAssetService(s3Mock, nil)
		ctx := context.Background()

		tmpKey := "tmp/VIDEO/x.mp4"

		s3Mock.On("CopyObject",
			mock.Anything, testBucket, tmpKey, mock.AnythingOfType("string"),
		).Return(nil)
		s3Mock.On("DeleteObject",
			mock.Anything, testBucket, tmpKey,
		).Return(nil)

		result, err := svc.FinalizeIfTemp(ctx, tmpKey)

		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(result, s3URL("assets/")))
		s3Mock.AssertExpectations(t)
	})

	t.Run("trusted CDN URL passes through unchanged", func(t *testing.T) {
		s3Mock := new(mockS3Client)
		svc := &AssetService{
			s3Client: s3Mock,
			bucket:   testBucket,
			region:   testRegion,
			cdnHost:  "cdn.example.com",
		}
		ctx := context.Background()

		cdnURL := "https://cdn.example.com/assets/VIDEO/x.mp4"
		result, err := svc.FinalizeIfTemp(ctx, cdnURL)

		require.NoError(t, err)
		assert.Equal(t, cdnURL, result)
		s3Mock.AssertNotCalled(t, "CopyObject")
	})

	t.Run("untrusted external URL passes through with warning (legacy CDN compat)", func(t *testing.T) {
		// FinalizeIfTemp no longer hard-rejects non-tmp untrusted URLs.
		// It logs a warning and returns the URL as-is so that existing products
		// carrying legacy CDN URLs (e.g. after a CDN_DOMAIN rotation) can still
		// be updated without errors. Handler-level validation guards fresh uploads.
		s3Mock := new(mockS3Client)
		svc := newTestAssetService(s3Mock, nil)
		ctx := context.Background()

		result, err := svc.FinalizeIfTemp(ctx, "https://evil.com/x.mp4")

		require.NoError(t, err)
		assert.Equal(t, "https://evil.com/x.mp4", result)
		s3Mock.AssertNotCalled(t, "CopyObject")
	})
}

// ---------------------------------------------------------------------------
// TestAssetService_DeleteAsset
// ---------------------------------------------------------------------------

func TestAssetService_DeleteAsset(t *testing.T) {
	tests := []struct {
		name        string
		assetURL    string
		deleteErr   error
		wantErr     bool
		errContains string
		errCode     errors.ErrorCode
	}{
		{
			name:     "successful delete",
			assetURL: s3URL("assets/IMAGE/2024/01/01/photo.jpg"),
		},
		{
			name:        "invalid URL prefix",
			assetURL:    "https://wrong-bucket.s3.amazonaws.com/assets/IMAGE/photo.jpg",
			wantErr:     true,
			errContains: "Invalid asset URL",
			errCode:     errors.ErrCodeBadRequest,
		},
		{
			name:        "URL not in assets/ prefix",
			assetURL:    s3URL("tmp/IMAGE/photo.jpg"),
			wantErr:     true,
			errContains: "Can only delete files in assets/ prefix",
			errCode:     errors.ErrCodeBadRequest,
		},
		{
			name:        "s3 delete error",
			assetURL:    s3URL("assets/IMAGE/2024/01/01/photo.jpg"),
			deleteErr:   fmt.Errorf("s3 error"),
			wantErr:     true,
			errContains: "Failed to delete asset",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s3Mock := new(mockS3Client)
			svc := newTestAssetService(s3Mock, nil)
			ctx := context.Background()

			// Only set up delete expectation if the URL passes validation
			if strings.HasPrefix(tc.assetURL, s3URL("")) {
				key := strings.TrimPrefix(tc.assetURL, s3URL(""))
				if strings.HasPrefix(key, assetsPrefix) {
					s3Mock.On("DeleteObject",
						mock.Anything, testBucket, key,
					).Return(tc.deleteErr)
				}
			}

			err := svc.DeleteAsset(ctx, tc.assetURL)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				if tc.errCode != "" {
					appErr, ok := errors.AsAppError(err)
					require.True(t, ok)
					assert.Equal(t, tc.errCode, appErr.Code)
				}
				return
			}

			require.NoError(t, err)
			s3Mock.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAssetService_GetUploadURL_Video
// ---------------------------------------------------------------------------

func TestAssetService_GetUploadURL_Video(t *testing.T) {
	tests := []struct {
		name        string
		req         domain.UploadAssetRequest
		expectError bool
		errContains string
	}{
		{
			name: "mp4 within 50MB succeeds",
			req: domain.UploadAssetRequest{
				FileName:    "demo.mp4",
				ContentType: "video/mp4",
				Size:        40 << 20, // 40 MB
				Type:        domain.AssetTypeVideo,
			},
			expectError: false,
		},
		{
			name: "mp4 over 50MB rejected",
			req: domain.UploadAssetRequest{
				FileName:    "demo.mp4",
				ContentType: "video/mp4",
				Size:        60 << 20, // 60 MB
				Type:        domain.AssetTypeVideo,
			},
			expectError: true,
			errContains: "size exceeds",
		},
		{
			name: "non-mp4 video rejected",
			req: domain.UploadAssetRequest{
				FileName:    "demo.webm",
				ContentType: "video/webm",
				Size:        10 << 20,
				Type:        domain.AssetTypeVideo,
			},
			expectError: true,
			errContains: "Invalid content type",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s3Mock := new(mockS3Client)
			s := newTestAssetService(s3Mock, nil)

			if !tc.expectError {
				s3Mock.On("GeneratePresignedPutURL",
					mock.Anything, testBucket, mock.AnythingOfType("string"),
					tc.req.ContentType, mock.AnythingOfType("time.Duration"),
				).Return("https://s3.presigned.url/put", nil)
			}

			_, err := s.GetUploadURL(context.Background(), tc.req)
			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
			s3Mock.AssertExpectations(t)
		})
	}
}

// ---------------------------------------------------------------------------
// TestHelpers_IsValidContentType
// ---------------------------------------------------------------------------

func TestHelpers_IsValidContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		assetType   domain.AssetType
		want        bool
	}{
		// IMAGE — valid
		{"image/jpeg is valid IMAGE", "image/jpeg", domain.AssetTypeImage, true},
		{"image/png is valid IMAGE", "image/png", domain.AssetTypeImage, true},
		{"image/webp is valid IMAGE", "image/webp", domain.AssetTypeImage, true},
		{"image/gif is valid IMAGE", "image/gif", domain.AssetTypeImage, true},

		// IMAGE — invalid
		{"video/mp4 is not valid IMAGE", "video/mp4", domain.AssetTypeImage, false},
		{"application/pdf is not valid IMAGE", contentTypePDF, domain.AssetTypeImage, false},

		// VIDEO — valid
		{"video/mp4 is valid VIDEO", "video/mp4", domain.AssetTypeVideo, true},

		// VIDEO — invalid
		{"video/webm is not valid VIDEO", "video/webm", domain.AssetTypeVideo, false},
		{"image/jpeg is not valid VIDEO", "image/jpeg", domain.AssetTypeVideo, false},
		{"application/pdf is not valid VIDEO", contentTypePDF, domain.AssetTypeVideo, false},

		// DOCUMENT — valid
		{"application/pdf is valid DOCUMENT", contentTypePDF, domain.AssetTypeDocument, true},
		{"application/msword is valid DOCUMENT", "application/msword", domain.AssetTypeDocument, true},
		{"docx is valid DOCUMENT", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", domain.AssetTypeDocument, true},
		{"application/vnd.ms-excel is valid DOCUMENT", "application/vnd.ms-excel", domain.AssetTypeDocument, true},
		{"xlsx is valid DOCUMENT", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", domain.AssetTypeDocument, true},
		{"text/csv is valid DOCUMENT", "text/csv", domain.AssetTypeDocument, true},

		// DOCUMENT — invalid
		{"image/png is not valid DOCUMENT", "image/png", domain.AssetTypeDocument, false},
		{"text/plain is not valid DOCUMENT", "text/plain", domain.AssetTypeDocument, false},

		// Unknown asset type
		{"unknown asset type returns false", "image/png", domain.AssetType("UNKNOWN"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isValidContentType(tc.contentType, tc.assetType)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestHelpers_GetMaxSize
// ---------------------------------------------------------------------------

func TestHelpers_GetMaxSize(t *testing.T) {
	tests := []struct {
		name      string
		assetType domain.AssetType
		want      int64
	}{
		{"IMAGE returns 50MB", domain.AssetTypeImage, maxImageSize},
		{"VIDEO returns 50MB", domain.AssetTypeVideo, maxVideoSize},
		{"DOCUMENT returns 10MB", domain.AssetTypeDocument, maxDocumentSize},
		{"unknown type defaults to 10MB", domain.AssetType("UNKNOWN"), maxDocumentSize},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := getMaxSize(tc.assetType)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// TestFinalizeUpload_DoesNotInvokeResizerForNonImage
// ---------------------------------------------------------------------------

func TestFinalizeUpload_DoesNotInvokeResizerForNonImage(t *testing.T) {
	s3Mock := new(mockS3Client)
	resizer := &mockResizer{}
	svc := newTestAssetService(s3Mock, resizer)

	tmpKey := "tmp/VIDEO/abc.mp4"
	parts := strings.SplitN(tmpKey, "/", 3)
	expectedFinalPrefix := fmt.Sprintf("assets/%s/%s/", parts[1], time.Now().Format("2006/01/02"))

	s3Mock.On("CopyObject",
		mock.Anything, testBucket, tmpKey,
		mock.MatchedBy(func(dstKey string) bool {
			return strings.HasPrefix(dstKey, expectedFinalPrefix)
		}),
	).Return(nil)
	s3Mock.On("DeleteObject",
		mock.Anything, testBucket, tmpKey,
	).Return(nil)

	_, err := svc.FinalizeUpload(context.Background(), tmpKey)
	if err != nil {
		t.Fatalf("FinalizeUpload returned error: %v", err)
	}
	if len(resizer.invocations) != 0 {
		t.Fatalf("expected 0 resizer invocations, got %d", len(resizer.invocations))
	}
}

// ---------------------------------------------------------------------------
// TestFinalizeUpload_InvokesResizerForImage
// ---------------------------------------------------------------------------

func TestFinalizeUpload_InvokesResizerForImage(t *testing.T) {
	s3Mock := new(mockS3Client)
	resizer := &mockResizer{}
	svc := newTestAssetService(s3Mock, resizer)

	tmpKey := "tmp/IMAGE/abc-123.jpg"
	parts := strings.SplitN(tmpKey, "/", 3)
	expectedFinalPrefix := fmt.Sprintf("assets/%s/%s/", parts[1], time.Now().Format("2006/01/02"))

	s3Mock.On("CopyObject",
		mock.Anything, testBucket, tmpKey,
		mock.MatchedBy(func(dstKey string) bool {
			return strings.HasPrefix(dstKey, expectedFinalPrefix)
		}),
	).Return(nil)
	s3Mock.On("DeleteObject",
		mock.Anything, testBucket, tmpKey,
	).Return(nil)

	_, err := svc.FinalizeUpload(context.Background(), tmpKey)
	if err != nil {
		t.Fatalf("FinalizeUpload returned error: %v", err)
	}

	if len(resizer.invocations) != 1 {
		t.Fatalf("expected 1 resizer invocation, got %d", len(resizer.invocations))
	}
	call := resizer.invocations[0]
	if call.FunctionName != "test-resizer-fn" {
		t.Errorf("function name = %q, want %q", call.FunctionName, "test-resizer-fn")
	}
	if !bytes.Contains(call.Payload, []byte(`"name":"test-bucket"`)) {
		t.Errorf("payload missing bucket: %s", call.Payload)
	}
	if !bytes.Contains(call.Payload, []byte(`"key":"assets/IMAGE/`)) {
		t.Errorf("payload missing key: %s", call.Payload)
	}
}

// ---------------------------------------------------------------------------
// TestFinalizeUpload_PropagatesResizerError
// ---------------------------------------------------------------------------

func TestFinalizeUpload_PropagatesResizerError(t *testing.T) {
	s3Mock := new(mockS3Client)
	resizer := &mockResizer{returnErr: fmt.Errorf("simulated lambda failure")}
	svc := newTestAssetService(s3Mock, resizer)

	tmpKey := "tmp/IMAGE/abc-123.jpg"
	parts := strings.SplitN(tmpKey, "/", 3)
	expectedFinalPrefix := fmt.Sprintf("assets/%s/%s/", parts[1], time.Now().Format("2006/01/02"))

	s3Mock.On("CopyObject",
		mock.Anything, testBucket, tmpKey,
		mock.MatchedBy(func(dstKey string) bool {
			return strings.HasPrefix(dstKey, expectedFinalPrefix)
		}),
	).Return(nil)

	_, err := svc.FinalizeUpload(context.Background(), tmpKey)
	s3Mock.AssertExpectations(t)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "image variants") {
		t.Errorf("error did not mention image variants: %v", err)
	}
}
