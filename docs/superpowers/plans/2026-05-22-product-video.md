# Product Video Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one MP4 video (with poster image) per product — uploaded from the admin portal, played in the storefront product page gallery.

**Architecture:** Two new nullable columns (`video_url`, `video_poster_url`) on the `products` PostgreSQL table. Reuse the existing tmp/→assets/ S3 asset flow (`AssetService.GetUploadURL` + `FinalizeIfTemp`). Storefront prepends a video item to the existing image gallery with an inline `<video controls>` player and a play-icon thumbnail overlay.

**Tech Stack:** Go 1.25 (chi, pgx, scany), AWS Lambda + S3, React 19 + TypeScript + react-hook-form + zod, Next.js 16.

**Spec:** `docs/superpowers/specs/2026-05-22-product-video-design.md`

---

## File Map

### Backend (`handloom-admin/`)
- **Create:** `migrations/004_product_video.sql`
- **Modify:** `internal/repository/postgres/columns.go` (add `ColVideoURL`, `ColVideoPosterURL`; extend `productColumns`)
- **Modify:** `internal/domain/entity.go` (add `VideoURL`, `VideoPosterURL` to `Product`; extend `NewProduct` and `ApplyUpdate`)
- **Modify:** `internal/domain/service.go` (add `VideoURL`, `VideoPosterURL` to `CreateProductRequest` + `UpdateProductRequest`)
- **Modify:** `internal/repository/postgres/product_repository.go` (extend `Create` + `Update` query builders)
- **Modify:** `internal/service/asset_service.go` (tighten `maxVideoSize` to 50 MB; restrict video content type to `video/mp4`)
- **Modify:** `internal/service/product_service.go` (finalize video + poster URLs on Create/Update; delete old video on URL change)
- **Modify:** `internal/handler/store/catalog_handler.go` (add `VideoURL`, `VideoPosterURL` to `StoreProduct`; extend `toStoreProduct`)
- **Test:** `internal/service/asset_service_test.go`
- **Test:** `internal/service/product_service_test.go`
- **Test:** `internal/repository/postgres/product_repository_test.go`

### Admin Frontend (`handloom-admin-frontend/`)
- **Modify:** `src/features/products/types.ts` (add `video_url`, `video_poster_url` to `Product`, `CreateProductRequest`)
- **Modify:** `src/features/products/components/ProductFormModal/ProductFormModal.tsx` (zod schema + form fields + submit payload)

### Storefront (`homechrome-store/`)
- **Modify:** `src/types/index.ts` (add `video_url`, `video_poster_url` to `Product`)
- **Modify:** `src/app/p/[slug]/ProductDetailView.tsx` (GalleryItem union, video render, JSON-LD)

---

## Conventions

- **Commit message format:** `feat(scope): short summary` — match existing repo style (e.g. `feat(catalog): add product video columns`).
- **Test commands:**
  - Backend single: `cd handloom-admin && go test -v -run TestName ./internal/service/...`
  - Backend full: `cd handloom-admin && make test`
  - Backend integration (Postgres): `cd handloom-admin && make test-integration` (requires LocalStack + Postgres running via `make setup-local`)
  - Admin FE: `cd handloom-admin-frontend && npm run test`
  - Storefront FE: `cd homechrome-store && npm test` (if present; otherwise `npm run build` smoke)
- **Lint before commit:** `golangci-lint run` (backend), `npm run lint` (frontends).
- **Wire / mocks:** No new interfaces added → no `make wire` / `make generate-mocks` needed.

---

## Task 1: PostgreSQL migration

**Files:**
- Create: `handloom-admin/migrations/004_product_video.sql`

- [ ] **Step 1: Write the migration**

Create `handloom-admin/migrations/004_product_video.sql`:

```sql
-- 004_product_video.sql
-- Add nullable video columns to products. Backwards compatible: existing rows
-- have NULL = no video.

ALTER TABLE products
  ADD COLUMN video_url        TEXT,
  ADD COLUMN video_poster_url TEXT;
```

- [ ] **Step 2: Apply locally and verify**

Run:
```bash
cd handloom-admin
make reset-db
```

Expected: migrator applies `001_*`, `002_*`, `003_*`, `004_product_video.sql` in order with no errors.

Verify columns exist:
```bash
docker exec -i $(docker ps -qf name=postgres) psql -U handloom -d handloom -c "\d products" | grep video
```
Expected output contains:
```
 video_url                 | text                        |
 video_poster_url          | text                        |
```

- [ ] **Step 3: Commit**

```bash
cd handloom-admin
git add migrations/004_product_video.sql
git commit -m "feat(catalog): add product video columns migration"
```

---

## Task 2: Column constants + `productColumns`

**Files:**
- Modify: `handloom-admin/internal/repository/postgres/columns.go`

- [ ] **Step 1: Add column constants**

In `handloom-admin/internal/repository/postgres/columns.go`, inside the "Product table columns" const block (currently ends at `ColSortOrder`), append two new constants:

```go
// Product table columns.
const (
	ColSKU                   = "sku"
	ColDescription           = "description"
	ColCategoryID            = "category_id"
	ColBasePrice             = "base_price"
	ColSellingPrice          = "selling_price"
	ColCostPrice             = "cost_price"
	ColCurrency              = "currency"
	ColDimLength             = "dim_length"
	ColDimWidth              = "dim_width"
	ColDimHeight             = "dim_height"
	ColDimUnit               = "dim_unit"
	ColWeight                = "weight"
	ColAllowCustomDimensions = "allow_custom_dimensions"
	ColPricingRuleID         = "pricing_rule_id"
	ColTags                  = "tags"
	ColSortOrder             = "sort_order"
	ColVideoURL              = "video_url"
	ColVideoPosterURL        = "video_poster_url"
)
```

- [ ] **Step 2: Extend `productColumns` slice**

In the same file, update the `productColumns` slice and its comment from "24 columns" to "26 columns":

```go
// productColumns lists the 26 columns selected for a product row.
var productColumns = []string{
	ColID, ColName, ColSlug, ColSKU, ColDescription, ColCategoryID,
	ColBasePrice, ColSellingPrice, ColCostPrice, ColCurrency,
	ColDimLength, ColDimWidth, ColDimHeight, ColDimUnit,
	ColWeight, ColAllowCustomDimensions, ColPricingRuleID,
	ColTags, ColStatus, ColSortOrder,
	ColVideoURL, ColVideoPosterURL,
	ColCreatedAt, ColUpdatedAt, ColCreatedBy, ColUpdatedBy,
}
```

- [ ] **Step 3: Build to verify**

```bash
cd handloom-admin
go build ./...
```

Expected: PASS with no errors. (Domain struct gets new fields in Task 3, but unused column constants are fine.)

- [ ] **Step 4: Commit**

```bash
git add internal/repository/postgres/columns.go
git commit -m "feat(catalog): add video column constants"
```

---

## Task 3: Domain entity + request structs

**Files:**
- Modify: `handloom-admin/internal/domain/entity.go`
- Modify: `handloom-admin/internal/domain/service.go`

- [ ] **Step 1: Add fields to `Product` struct**

In `handloom-admin/internal/domain/entity.go` inside `type Product struct {...}`, locate the "Media (stored in product_images table)" block:

```go
	// Media (stored in product_images table)
	Images []ProductImage `json:"images,omitempty"`
```

Replace with:

```go
	// Media (stored in product_images table)
	Images []ProductImage `json:"images,omitempty"`

	// Video (stored on products row; one optional video per product)
	VideoURL       string `json:"video_url,omitempty" db:"video_url"`
	VideoPosterURL string `json:"video_poster_url,omitempty" db:"video_poster_url"`
```

- [ ] **Step 2: Add fields to `CreateProductRequest`**

In `handloom-admin/internal/domain/service.go` inside `type CreateProductRequest struct {...}`, locate the `Images` field:

```go
	Images                []ProductImage         `json:"images,omitempty"`
```

Replace with:

```go
	Images                []ProductImage         `json:"images,omitempty"`
	VideoURL              string                 `json:"video_url,omitempty"`
	VideoPosterURL        string                 `json:"video_poster_url,omitempty"`
```

- [ ] **Step 3: Add fields to `UpdateProductRequest`**

In `handloom-admin/internal/domain/service.go` inside `type UpdateProductRequest struct {...}`, locate the `Images` field:

```go
	Images                []ProductImage         `json:"images,omitempty"`
```

Replace with:

```go
	Images                []ProductImage         `json:"images,omitempty"`
	VideoURL              *string                `json:"video_url,omitempty"`
	VideoPosterURL        *string                `json:"video_poster_url,omitempty"`
```

(Pointer types because update fields are partial; nil = don't change, empty string = clear.)

- [ ] **Step 4: Extend `NewProduct` constructor**

In `handloom-admin/internal/domain/entity.go` inside `func NewProduct(...)`, locate:

```go
		Images:                req.Images,
```

Add immediately after:

```go
		VideoURL:              req.VideoURL,
		VideoPosterURL:        req.VideoPosterURL,
```

- [ ] **Step 5: Extend `ApplyUpdate` method**

In `handloom-admin/internal/domain/entity.go` inside `func (p *Product) ApplyUpdate(req UpdateProductRequest)`, locate the end of the existing field block (just before the closing `}` of the function). Add the following block before the closing brace:

```go
	if req.VideoURL != nil {
		p.VideoURL = *req.VideoURL
	}
	if req.VideoPosterURL != nil {
		p.VideoPosterURL = *req.VideoPosterURL
	}
```

- [ ] **Step 6: Build**

```bash
cd handloom-admin
go build ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/entity.go internal/domain/service.go
git commit -m "feat(catalog): add VideoURL + VideoPosterURL to product domain"
```

---

## Task 4: Repository Create + Update column writes

**Files:**
- Modify: `handloom-admin/internal/repository/postgres/product_repository.go`

- [ ] **Step 1: Extend `Create` query builder**

In `handloom-admin/internal/repository/postgres/product_repository.go` inside `func (r *ProductRepository) Create(...)`, locate the `qb := querybuilder.Insert("products")` block. After:

```go
		Set(ColSortOrder, product.SortOrder).
```

Add:

```go
		Set(ColVideoURL, nullableString(product.VideoURL)).
		Set(ColVideoPosterURL, nullableString(product.VideoPosterURL)).
```

- [ ] **Step 2: Extend `Update` query builder**

In the same file inside `func (r *ProductRepository) Update(...)`, locate the `qb := querybuilder.Update("products")` block. After:

```go
		Set(ColSortOrder, product.SortOrder).
```

Add:

```go
		Set(ColVideoURL, nullableString(product.VideoURL)).
		Set(ColVideoPosterURL, nullableString(product.VideoPosterURL)).
```

- [ ] **Step 3: Add `nullableString` helper**

At the bottom of `handloom-admin/internal/repository/postgres/product_repository.go`, add:

```go
// nullableString returns nil for empty strings so that empty values are written
// as NULL rather than empty TEXT. Avoids querying `WHERE col = ''` ambiguity.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
```

- [ ] **Step 4: Build**

```bash
cd handloom-admin
go build ./...
```

Expected: PASS.

- [ ] **Step 5: Write integration test for round-trip**

Locate `handloom-admin/internal/repository/postgres/product_repository_test.go`. Find the existing helper that creates a fixture product (e.g. a function returning a `*domain.Product` used by other tests) and add a new test:

```go
func TestProductRepository_VideoRoundTrip(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewProductRepository(pool)
	category := seedCategory(t, pool)

	t.Run("create with video round-trips", func(t *testing.T) {
		p := newTestProduct(category.ID)
		p.VideoURL = "https://cdn.example.com/assets/VIDEO/2026/05/22/uuid.mp4"
		p.VideoPosterURL = "https://cdn.example.com/assets/IMAGE/2026/05/22/poster.jpg"

		err := repo.Create(context.Background(), p, nil)
		require.NoError(t, err)

		got, err := repo.GetByID(context.Background(), p.ID)
		require.NoError(t, err)
		require.Equal(t, p.VideoURL, got.VideoURL)
		require.Equal(t, p.VideoPosterURL, got.VideoPosterURL)
	})

	t.Run("nil video URLs persist as empty strings", func(t *testing.T) {
		p := newTestProduct(category.ID)

		err := repo.Create(context.Background(), p, nil)
		require.NoError(t, err)

		got, err := repo.GetByID(context.Background(), p.ID)
		require.NoError(t, err)
		require.Empty(t, got.VideoURL)
		require.Empty(t, got.VideoPosterURL)
	})

	t.Run("update clears video URLs", func(t *testing.T) {
		p := newTestProduct(category.ID)
		p.VideoURL = "https://cdn.example.com/assets/VIDEO/old.mp4"
		require.NoError(t, repo.Create(context.Background(), p, nil))

		p.VideoURL = ""
		p.VideoPosterURL = ""
		require.NoError(t, repo.Update(context.Background(), p))

		got, err := repo.GetByID(context.Background(), p.ID)
		require.NoError(t, err)
		require.Empty(t, got.VideoURL)
		require.Empty(t, got.VideoPosterURL)
	})
}
```

If `newTestProduct` / `seedCategory` / `newTestPool` helpers don't exist with those exact names, copy the pattern from the nearest existing test in the same file and adapt — the goal is the three sub-tests above against the real repository.

- [ ] **Step 6: Run integration tests**

```bash
cd handloom-admin
make setup-local            # starts LocalStack + Postgres if not running
go test -v -run TestProductRepository_VideoRoundTrip ./internal/repository/postgres/...
```

Expected: PASS for all three sub-tests.

- [ ] **Step 7: Run full repo test suite to confirm no regression**

```bash
go test -v ./internal/repository/postgres/...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/repository/postgres/product_repository.go internal/repository/postgres/product_repository_test.go
git commit -m "feat(catalog): persist product video columns in repo"
```

---

## Task 5: Tighten asset service video constraints

**Files:**
- Modify: `handloom-admin/internal/service/asset_service.go`
- Modify: `handloom-admin/internal/service/asset_service_test.go`

- [ ] **Step 1: Write failing test first**

In `handloom-admin/internal/service/asset_service_test.go`, add a new test (do not delete or rename existing tests):

```go
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
			s := service.NewAssetService(newFakeS3Client(t), "test-bucket", "ap-south-1", "", "")
			_, err := s.GetUploadURL(context.Background(), tc.req)
			if tc.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
		})
	}
}
```

If a fake/mock S3 client helper named `newFakeS3Client` doesn't already exist, look for the existing setup pattern in the same test file (search for `NewAssetService(`) and reuse the same fixture — copy that helper or adapt its name.

- [ ] **Step 2: Run test, see it fail**

```bash
cd handloom-admin
go test -v -run TestAssetService_GetUploadURL_Video ./internal/service/...
```

Expected: FAIL — "mp4 over 50MB rejected" passes because current cap is 100MB but request is 60MB so currently no error (wait, 60MB < 100MB so it succeeds → test fails). "non-mp4 video rejected" fails because the current code accepts any `video/*` prefix.

- [ ] **Step 3: Tighten `maxVideoSize`**

In `handloom-admin/internal/service/asset_service.go`, change the const block (lines ~19–28):

```go
const (
	presignExpiry   = 15 * time.Minute
	maxImageSize    = 50 << 20  // 50 MB
	maxVideoSize    = 50 << 20  // 50 MB
	maxDocumentSize = 10 << 20  // 10 MB
	tmpPrefix       = "tmp/"
	assetsPrefix    = "assets/"

	contentTypePDF = "application/pdf"

	videoMP4ContentType = "video/mp4"
)
```

(Changed `maxVideoSize` from `100 << 20` to `50 << 20`. Added `videoMP4ContentType` constant.)

- [ ] **Step 4: Restrict video content type to mp4**

In `handloom-admin/internal/service/asset_service.go` inside `func isValidContentType(...)`, update the `domain.AssetTypeVideo` case:

```go
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
```

- [ ] **Step 5: Run test, see it pass**

```bash
go test -v -run TestAssetService_GetUploadURL_Video ./internal/service/...
```

Expected: PASS for all three sub-tests.

- [ ] **Step 6: Run full asset service tests to confirm no regression**

```bash
go test -v ./internal/service/...
```

Expected: PASS. If any pre-existing video test relied on `webm`/`mov`, update it to use `video/mp4` — these tests would have been mis-aligned with the new restricted policy.

- [ ] **Step 7: Commit**

```bash
git add internal/service/asset_service.go internal/service/asset_service_test.go
git commit -m "feat(asset): restrict video uploads to mp4 + 50MB"
```

---

## Task 6: Product service finalize + delete-on-replace

**Files:**
- Modify: `handloom-admin/internal/service/product_service.go`
- Modify: `handloom-admin/internal/service/product_service_test.go`

Domain interface check: `domain.AssetFinalizer` currently has only `FinalizeIfTemp`. We also need delete on replace. Inspect the interface in `internal/domain/service.go` (search `AssetFinalizer interface`) — if it lacks `DeleteAsset`, add it before this task. Step 1 below does that.

- [ ] **Step 1: Extend `AssetFinalizer` interface**

In `handloom-admin/internal/domain/service.go`, search for `AssetFinalizer interface`. Replace the interface with:

```go
type AssetFinalizer interface {
	FinalizeIfTemp(ctx context.Context, value string) (string, error)
	DeleteAsset(ctx context.Context, assetURL string) error
}
```

`AssetService` already implements `DeleteAsset` (see `asset_service.go` line ~191), so the interface assertion `var _ domain.AssetFinalizer = (*AssetService)(nil)` will continue to hold.

- [ ] **Step 2: Regenerate mocks**

```bash
cd handloom-admin
make generate-mocks
```

Expected: `internal/mocks/asset_domain_mock.go` updated with `DeleteAsset` method.

- [ ] **Step 3: Write failing test for Create finalize**

In `handloom-admin/internal/service/product_service_test.go`, add:

```go
func TestProductService_Create_FinalizesVideoAndPoster(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	productRepo := mocks.NewMockProductRepository(ctrl)
	categoryRepo := mocks.NewMockCategoryRepository(ctrl)
	inventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	finalizer := mocks.NewMockAssetFinalizer(ctrl)
	publisher := newNoopPublisher()

	svc := service.NewProductService(productRepo, categoryRepo, inventoryRepo, finalizer, publisher)

	categoryRepo.EXPECT().GetByID(gomock.Any(), "cat1").
		Return(&domain.Category{ID: "cat1"}, nil)

	finalizer.EXPECT().FinalizeIfTemp(gomock.Any(), "tmp/VIDEO/uuid.mp4").
		Return("https://cdn.example.com/assets/VIDEO/2026/05/22/uuid.mp4", nil)
	finalizer.EXPECT().FinalizeIfTemp(gomock.Any(), "tmp/IMAGE/poster.jpg").
		Return("https://cdn.example.com/assets/IMAGE/2026/05/22/poster.jpg", nil)

	productRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p *domain.Product, _ *domain.Inventory) error {
			require.Equal(t, "https://cdn.example.com/assets/VIDEO/2026/05/22/uuid.mp4", p.VideoURL)
			require.Equal(t, "https://cdn.example.com/assets/IMAGE/2026/05/22/poster.jpg", p.VideoPosterURL)
			return nil
		})
	categoryRepo.EXPECT().IncrementProductCount(gomock.Any(), "cat1", 1).Return(nil)

	_, err := svc.Create(context.Background(), domain.CreateProductRequest{
		Name:           "Test",
		SKU:            "SKU-1",
		CategoryID:     "cat1",
		BasePrice:      100,
		SellingPrice:   100,
		VideoURL:       "tmp/VIDEO/uuid.mp4",
		VideoPosterURL: "tmp/IMAGE/poster.jpg",
	}, "user-1")
	require.NoError(t, err)
}
```

If the existing tests use a different mock factory pattern (search for `mocks.NewMockProductRepository` in the same file to confirm naming), match their style. `newNoopPublisher` should follow the pattern already used by other product service tests for the publisher dependency.

- [ ] **Step 4: Run test, see it fail**

```bash
cd handloom-admin
go test -v -run TestProductService_Create_FinalizesVideoAndPoster ./internal/service/...
```

Expected: FAIL — `FinalizeIfTemp` not called for video / poster URLs (current code only iterates `req.Images`).

- [ ] **Step 5: Implement Create finalize**

In `handloom-admin/internal/service/product_service.go` inside `func (s *ProductService) Create(...)`, locate:

```go
	// Finalize any tmp/ image keys to permanent assets/ URLs
	for i, img := range req.Images {
		finalURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, img.URL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize image")
		}
		req.Images[i].URL = finalURL
	}
```

Add immediately after that block:

```go
	if req.VideoURL != "" {
		finalURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, req.VideoURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize video")
		}
		req.VideoURL = finalURL
	}
	if req.VideoPosterURL != "" {
		finalURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, req.VideoPosterURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize video poster")
		}
		req.VideoPosterURL = finalURL
	}
```

- [ ] **Step 6: Run Create test, see it pass**

```bash
go test -v -run TestProductService_Create_FinalizesVideoAndPoster ./internal/service/...
```

Expected: PASS.

- [ ] **Step 7: Write failing test for Update finalize + delete-on-replace**

In the same test file:

```go
func TestProductService_Update_ReplacesVideoAndDeletesOld(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	productRepo := mocks.NewMockProductRepository(ctrl)
	categoryRepo := mocks.NewMockCategoryRepository(ctrl)
	inventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	finalizer := mocks.NewMockAssetFinalizer(ctrl)
	publisher := newNoopPublisher()

	svc := service.NewProductService(productRepo, categoryRepo, inventoryRepo, finalizer, publisher)

	existing := &domain.Product{
		ID:             "p1",
		CategoryID:     "cat1",
		VideoURL:       "https://cdn.example.com/assets/VIDEO/old.mp4",
		VideoPosterURL: "https://cdn.example.com/assets/IMAGE/old.jpg",
	}
	productRepo.EXPECT().GetByID(gomock.Any(), "p1").Return(existing, nil)
	categoryRepo.EXPECT().GetByID(gomock.Any(), "cat1").Return(&domain.Category{ID: "cat1"}, nil)

	finalizer.EXPECT().FinalizeIfTemp(gomock.Any(), "tmp/VIDEO/new.mp4").
		Return("https://cdn.example.com/assets/VIDEO/new.mp4", nil)
	finalizer.EXPECT().DeleteAsset(gomock.Any(), "https://cdn.example.com/assets/VIDEO/old.mp4").Return(nil)

	productRepo.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p *domain.Product) error {
			require.Equal(t, "https://cdn.example.com/assets/VIDEO/new.mp4", p.VideoURL)
			return nil
		})

	newVideo := "tmp/VIDEO/new.mp4"
	_, err := svc.Update(context.Background(), "p1", domain.UpdateProductRequest{
		VideoURL: &newVideo,
	}, "user-1")
	require.NoError(t, err)
}
```

- [ ] **Step 8: Run, see it fail**

```bash
go test -v -run TestProductService_Update_ReplacesVideoAndDeletesOld ./internal/service/...
```

Expected: FAIL.

- [ ] **Step 9: Implement Update finalize + delete-on-replace**

In `handloom-admin/internal/service/product_service.go` inside `func (s *ProductService) Update(...)`, locate:

```go
	// Finalize any tmp/ image keys in the update request
	for i, img := range req.Images {
		finalURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, img.URL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize image")
		}
		req.Images[i].URL = finalURL
	}
```

Add immediately after:

```go
	if req.VideoURL != nil {
		oldURL := product.VideoURL
		newURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, *req.VideoURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize video")
		}
		*req.VideoURL = newURL
		if oldURL != "" && oldURL != newURL {
			if delErr := s.assetFinalizer.DeleteAsset(ctx, oldURL); delErr != nil {
				slog.WarnContext(ctx, "Failed to delete old product video", "url", oldURL, "error", delErr)
			}
		}
	}
	if req.VideoPosterURL != nil {
		oldURL := product.VideoPosterURL
		newURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, *req.VideoPosterURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize video poster")
		}
		*req.VideoPosterURL = newURL
		if oldURL != "" && oldURL != newURL {
			if delErr := s.assetFinalizer.DeleteAsset(ctx, oldURL); delErr != nil {
				slog.WarnContext(ctx, "Failed to delete old product video poster", "url", oldURL, "error", delErr)
			}
		}
	}
```

- [ ] **Step 10: Run Update test, see it pass**

```bash
go test -v -run TestProductService_Update_ReplacesVideoAndDeletesOld ./internal/service/...
```

Expected: PASS.

- [ ] **Step 11: Run full product service suite**

```bash
go test -v ./internal/service/...
```

Expected: PASS (no regressions).

- [ ] **Step 12: Lint**

```bash
golangci-lint run
```

Expected: PASS.

- [ ] **Step 13: Commit**

```bash
git add internal/domain/service.go internal/mocks/asset_domain_mock.go internal/service/product_service.go internal/service/product_service_test.go
git commit -m "feat(product): finalize video + delete old on replace"
```

---

## Task 7: Store catalog DTO exposes video

**Files:**
- Modify: `handloom-admin/internal/handler/store/catalog_handler.go`
- Modify: `handloom-admin/internal/handler/store/catalog_handler_test.go` (if it exists; create test inline if not)

- [ ] **Step 1: Add fields to `StoreProduct`**

In `handloom-admin/internal/handler/store/catalog_handler.go` inside `type StoreProduct struct {...}`, locate:

```go
	// Media
	Images []domain.ProductImage `json:"images,omitempty"`
```

Replace with:

```go
	// Media
	Images         []domain.ProductImage `json:"images,omitempty"`
	VideoURL       string                `json:"video_url,omitempty"`
	VideoPosterURL string                `json:"video_poster_url,omitempty"`
```

- [ ] **Step 2: Extend `toStoreProduct`**

In the same file inside `func toStoreProduct(p *domain.Product) *StoreProduct`, locate:

```go
		Images:                p.Images,
```

Add immediately after:

```go
		VideoURL:              p.VideoURL,
		VideoPosterURL:        p.VideoPosterURL,
```

- [ ] **Step 3: Write/extend test**

Add (or extend an existing) test in `handloom-admin/internal/handler/store/catalog_handler_test.go`:

```go
func TestToStoreProduct_IncludesVideo(t *testing.T) {
	p := &domain.Product{
		ID:             "p1",
		VideoURL:       "https://cdn.example.com/assets/VIDEO/x.mp4",
		VideoPosterURL: "https://cdn.example.com/assets/IMAGE/x.jpg",
	}
	sp := store.ToStoreProduct(p) // adjust if helper is unexported — call via http test instead
	require.Equal(t, p.VideoURL, sp.VideoURL)
	require.Equal(t, p.VideoPosterURL, sp.VideoPosterURL)
}
```

If `toStoreProduct` is unexported and the package has no test-export shim, write the test inside the same package (`package store`) so it can call the private function directly. Otherwise drive it via an HTTP handler test (look for the nearest existing handler test in the same dir and copy the pattern).

- [ ] **Step 4: Run**

```bash
cd handloom-admin
go test -v ./internal/handler/store/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/store/catalog_handler.go internal/handler/store/catalog_handler_test.go
git commit -m "feat(store): expose product video fields in public API"
```

---

## Task 8: Admin frontend types

**Files:**
- Modify: `handloom-admin-frontend/src/features/products/types.ts`

- [ ] **Step 1: Add fields to `Product`**

In `handloom-admin-frontend/src/features/products/types.ts` inside `export interface Product { ... }`, locate:

```ts
  images?: ProductImage[];
```

Add immediately after:

```ts
  video_url?: string;
  video_poster_url?: string;
```

- [ ] **Step 2: Add fields to `CreateProductRequest`**

In the same file inside `export interface CreateProductRequest { ... }`, locate:

```ts
  images?: ProductImage[];
```

Add immediately after:

```ts
  video_url?: string;
  video_poster_url?: string;
```

- [ ] **Step 3: Type-check**

```bash
cd handloom-admin-frontend
npm run typecheck
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add src/features/products/types.ts
git commit -m "feat(products): add video fields to admin types"
```

---

## Task 9: Admin product form video upload UI

**Files:**
- Modify: `handloom-admin-frontend/src/features/products/components/ProductFormModal/ProductFormModal.tsx`

The existing `ImageUpload` component already supports video via `accept` prop and bypasses image compression for videos (see `src/shared/components/ui/ImageUpload.tsx:31` — `getAssetType` returns `'VIDEO'` for `video/*`). We reuse it for the video slot in single-file mode, and again for the poster slot in image-only mode.

- [ ] **Step 1: Add fields to zod schema**

In `handloom-admin-frontend/src/features/products/components/ProductFormModal/ProductFormModal.tsx`, locate the `productSchema` zod block:

```ts
const productSchema = z.object({
  // ...
  images: z.array(z.string()).optional(),
});
```

Replace the final `images` line with:

```ts
  images: z.array(z.string()).optional(),
  video_url: z.string().optional().or(z.literal('')),
  video_poster_url: z.string().optional().or(z.literal('')),
});
```

(URL/tmp-key string; empty means no video. Not `.url()` because tmp keys like `tmp/VIDEO/uuid.mp4` aren't valid URLs.)

- [ ] **Step 2: Add defaults to both `reset({...})` calls**

In the same file, both `reset({...})` blocks (around the edit branch and the create branch in the `useEffect` that resets the form) end with `images: ...`. After `images: ...,` in BOTH blocks, append:

```ts
          video_url: product.video_url || '',
          video_poster_url: product.video_poster_url || '',
```

For the create branch (where `product` is undefined), use empty strings:

```ts
          video_url: '',
          video_poster_url: '',
```

Also append the defaults to the initial `useForm({ defaultValues: {...} })` block at the top of the component (line ~85). Find the existing `images: [],` line and after it add:

```ts
      video_url: '',
      video_poster_url: '',
```

- [ ] **Step 3: Add fields to submit payload**

In the `onSubmit` function inside `const requestData: CreateProductRequest = { ... }`, locate the final `attributes:` line. After the existing `images: (data.images || []).map(...),` block, add:

```ts
      video_url: data.video_url || undefined,
      video_poster_url: data.video_poster_url || undefined,
```

(Both at the same indentation as `images`. `|| undefined` so empty strings don't get sent.)

- [ ] **Step 4: Render upload widgets**

Locate the `<Controller name="images" ...>` block (line ~410). After the closing `</Controller>` wrapping div (the `<div className="md:col-span-2">` that contains the images uploader), add a new section before the closing `</div>` of "Basic Information":

```tsx
            <div className="md:col-span-2">
              <Controller
                name="video_url"
                control={control}
                render={({ field }) => (
                  <ImageUpload
                    label="Product Video (optional)"
                    value={field.value || ''}
                    onChange={(value) =>
                      field.onChange(Array.isArray(value) ? value[0] || '' : value)
                    }
                    accept="video/mp4"
                    maxSizeMB={50}
                    hint="MP4 only, up to 50MB. One video per product."
                    error={errors.video_url?.message}
                  />
                )}
              />
            </div>

            <div className="md:col-span-2">
              <Controller
                name="video_poster_url"
                control={control}
                render={({ field }) => (
                  <ImageUpload
                    label="Video Poster Image (optional)"
                    value={field.value || ''}
                    onChange={(value) =>
                      field.onChange(Array.isArray(value) ? value[0] || '' : value)
                    }
                    accept="image/*"
                    maxSizeMB={2}
                    hint="Thumbnail shown before the video plays. Recommended 16:9."
                    error={errors.video_poster_url?.message}
                  />
                )}
              />
            </div>
```

- [ ] **Step 5: Type-check + lint**

```bash
cd handloom-admin-frontend
npm run typecheck
npm run lint
```

Expected: PASS.

- [ ] **Step 6: Manual smoke**

Start the dev server pointing to local backend:

```bash
cd handloom-admin
make setup-local
make run
```

In another terminal:

```bash
cd handloom-admin-frontend
npm run dev:local
```

Open `http://localhost:5173`, log in, navigate to Products, open Create modal. Verify:
1. Two new upload widgets appear ("Product Video" and "Video Poster Image") below the image uploader.
2. Picking a small MP4 (e.g. < 5MB sample from `https://samplelib.com/sample-mp4.html`) triggers a presigned upload — toast says "File uploaded successfully" and a preview video appears.
3. Picking a > 50MB file shows a toast error "exceeds 50MB limit".
4. Picking a `.webm` file shows the toast error "exceeds" or the upload fails at the backend with 400.
5. Saving the product persists `video_url` (check via `docker exec` query: `SELECT video_url FROM products WHERE id = '<id>';`).

Report any visible issue before committing.

- [ ] **Step 7: Commit**

```bash
git add src/features/products/components/ProductFormModal/ProductFormModal.tsx
git commit -m "feat(products): add video + poster upload widgets to admin form"
```

---

## Task 10: Storefront types

**Files:**
- Modify: `homechrome-store/src/types/index.ts`

- [ ] **Step 1: Add fields to `Product`**

In `homechrome-store/src/types/index.ts` inside `export interface Product { ... }`, locate:

```ts
  images: ProductImage[];
```

Add immediately after:

```ts
  video_url?: string;
  video_poster_url?: string;
```

- [ ] **Step 2: Build**

```bash
cd homechrome-store
npm run build
```

Expected: PASS (no usage yet but type compiles).

- [ ] **Step 3: Commit**

```bash
git add src/types/index.ts
git commit -m "feat(store): add video fields to Product type"
```

---

## Task 11: Storefront gallery with video

**Files:**
- Modify: `homechrome-store/src/app/p/[slug]/ProductDetailView.tsx`

- [ ] **Step 1: Refactor gallery state to support video**

In `homechrome-store/src/app/p/[slug]/ProductDetailView.tsx`, replace lines ~25–33 (the existing image gallery state) with:

```tsx
  const sortedImages = [...(product.images || [])].sort((a, b) => {
    if (a.is_primary && !b.is_primary) return -1;
    if (!a.is_primary && b.is_primary) return 1;
    return a.sort_order - b.sort_order;
  });

  type GalleryItem =
    | { kind: 'video'; url: string; poster?: string }
    | { kind: 'image'; image: ProductImage };

  const galleryItems: GalleryItem[] = [
    ...(product.video_url
      ? [{ kind: 'video' as const, url: product.video_url, poster: product.video_poster_url }]
      : []),
    ...sortedImages.map((image) => ({ kind: 'image' as const, image })),
  ];

  const [selectedItem, setSelectedItem] = useState<GalleryItem | null>(
    galleryItems[0] || null,
  );
```

Delete the now-unused `selectedImage` / `setSelectedImage` declarations.

- [ ] **Step 2: Update main viewer**

Replace the existing `{/* Main Image */}` block (around lines ~121–137) with:

```tsx
          {/* Main Media */}
          <div className="relative aspect-square overflow-hidden rounded-xl bg-gray-100">
            {selectedItem?.kind === 'video' ? (
              <video
                key={selectedItem.url}
                src={selectedItem.url}
                controls
                playsInline
                preload="metadata"
                poster={selectedItem.poster}
                className="h-full w-full object-contain"
              />
            ) : selectedItem?.kind === 'image' ? (
              <Image
                src={selectedItem.image.url}
                alt={selectedItem.image.alt_text || product.name}
                fill
                sizes="(max-width: 1024px) 100vw, 50vw"
                className="object-cover"
                priority
              />
            ) : (
              <div className="flex h-full items-center justify-center bg-primary-light/30">
                <PhotoIcon className="h-16 w-16 text-primary/40" />
              </div>
            )}
          </div>
```

- [ ] **Step 3: Update thumbnail strip**

Replace the existing thumbnails block (lines ~140–163) with:

```tsx
          {/* Thumbnails */}
          {galleryItems.length > 1 && (
            <div className="mt-4 flex gap-3 overflow-x-auto pb-2">
              {galleryItems.map((item, index) => {
                const isActive = selectedItem === item;
                const baseCls = `relative h-20 w-20 flex-shrink-0 overflow-hidden rounded-lg border-2 transition-colors ${
                  isActive ? 'border-primary' : 'border-transparent hover:border-border'
                }`;

                if (item.kind === 'video') {
                  return (
                    <button
                      key="video"
                      type="button"
                      onClick={() => setSelectedItem(item)}
                      className={baseCls}
                      aria-label="Play product video"
                    >
                      {item.poster ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={item.poster}
                          alt="Product video thumbnail"
                          className="h-full w-full object-cover"
                        />
                      ) : (
                        <div className="h-full w-full bg-gray-200" />
                      )}
                      <span className="absolute inset-0 flex items-center justify-center bg-black/30">
                        <PlayIcon className="h-6 w-6 text-white" />
                      </span>
                    </button>
                  );
                }

                const img = item.image;
                return (
                  <button
                    key={`img-${index}`}
                    type="button"
                    onClick={() => setSelectedItem(item)}
                    className={baseCls}
                  >
                    <Image
                      src={img.url}
                      alt={img.alt_text || `${product.name} ${index + 1}`}
                      fill
                      sizes="80px"
                      className="object-cover"
                    />
                  </button>
                );
              })}
            </div>
          )}
```

- [ ] **Step 4: Add `PlayIcon` import**

At the top of the file, locate:

```tsx
import { PhotoIcon } from '@heroicons/react/24/outline';
```

Add a separate import line right below it for the solid play icon (filled icon reads better as a play overlay):

```tsx
import { PhotoIcon } from '@heroicons/react/24/outline';
import { PlayIcon } from '@heroicons/react/24/solid';
```

- [ ] **Step 5: Add JSON-LD VideoObject**

Find the `ProductDetailView` page's JSON-LD output. It's most likely in `homechrome-store/src/app/p/[slug]/page.tsx` (the server component that renders `ProductDetailView`). Open that file and locate the JSON-LD `<script>` block (search for `application/ld+json` or `@type": "Product"`). Inside the product schema object, conditionally add:

```ts
...(product.video_url
  ? {
      video: {
        '@type': 'VideoObject',
        name: product.name,
        thumbnailUrl: product.video_poster_url ?? product.images?.[0]?.url,
        contentUrl: product.video_url,
        uploadDate: product.updated_at,
      },
    }
  : {}),
```

If the JSON-LD lives elsewhere or uses a helper, follow that helper's pattern instead — the goal is one `VideoObject` linked from the `Product` schema when `video_url` is set.

- [ ] **Step 6: Build**

```bash
cd homechrome-store
npm run build
```

Expected: PASS.

- [ ] **Step 7: Manual smoke**

```bash
cd homechrome-store
npm run dev
```

Steps:
1. Create a product via the admin UI with a video + poster.
2. Open the storefront product page at `http://localhost:3000/p/<slug>`.
3. Verify the first thumbnail shows the poster image with a play-icon overlay.
4. Click it → main viewer shows `<video controls>`. Press play → video plays inline (no fullscreen hijack on mobile).
5. Click an image thumbnail → switches back to `<Image>`.
6. Verify a product *without* a video still renders only images (no broken first slot).
7. View page source → confirm `"@type": "VideoObject"` block present.

- [ ] **Step 8: Commit**

```bash
git add src/app/p/[slug]/ProductDetailView.tsx src/app/p/[slug]/page.tsx
git commit -m "feat(store): show product video in detail gallery"
```

---

## Task 12: End-to-end smoke (no code)

- [ ] **Step 1: Full backend test sweep**

```bash
cd handloom-admin
make test
golangci-lint run
```

Expected: PASS.

- [ ] **Step 2: Full admin frontend check**

```bash
cd handloom-admin-frontend
npm run check
```

Expected: PASS (typecheck + lint + format:check).

- [ ] **Step 3: Storefront build**

```bash
cd homechrome-store
npm run build
```

Expected: PASS.

- [ ] **Step 4: E2E manual flow**

1. `cd handloom-admin && make reset-db && make run` (rebuilds with migration 004 applied).
2. `cd handloom-admin-frontend && npm run dev:local`.
3. `cd homechrome-store && npm run dev`.
4. Admin: create new product → upload video (sample 5MB MP4) + poster (sample JPG) + save.
5. Admin: open the same product → confirm both URLs populated, preview renders.
6. Admin: edit, swap video for a different MP4 → save. Confirm: old S3 object deleted via container logs (LocalStack S3 access log), new URL stored.
7. Storefront: open product page → video thumb appears first with play icon. Click + play succeeds.
8. Storefront: open page source → `VideoObject` JSON-LD present.
9. Storefront: open a video-less product → gallery shows only images, no broken state.

If all steps pass, feature is complete.

---

## Self-Review Notes (pre-execution)

Spec coverage:
- ✅ Data model (migration, columns) — Tasks 1–2
- ✅ Domain entity + requests — Task 3
- ✅ Repo persistence — Task 4
- ✅ Asset constraints (50MB / mp4 only) — Task 5
- ✅ Service finalize + delete-on-replace — Task 6
- ✅ Store DTO exposure — Task 7
- ✅ Admin types — Task 8
- ✅ Admin upload UI — Task 9
- ✅ Storefront types — Task 10
- ✅ Storefront gallery + JSON-LD — Task 11
- ✅ E2E manual smoke — Task 12

Potential follow-ups (not in scope here):
- Track video play events in storefront analytics (out of scope per spec).
- Auto-poster extraction via ffmpeg Lambda (out of scope).
- Multi-video support (would require switching from columns to a `product_videos` table — defer until limit grows).
