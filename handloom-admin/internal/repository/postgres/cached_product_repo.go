package postgres

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/handloom/admin/internal/cache"
	"github.com/handloom/admin/internal/domain"
)

const (
	prodItemTTL    = 1 * time.Hour
	prodAttrTTL    = 1 * time.Hour
	prodListTTL    = 1 * time.Hour
	prodPrefix     = "prod:"
	prodListPrefix = "prod:list:"
)

func prodKey(id string) string          { return fmt.Sprintf("prod:%s", id) }
func prodSKUKey(sku string) string      { return fmt.Sprintf("prod:sku:%s", sku) }
func prodCatPrefix(catID string) string { return fmt.Sprintf("prod:cat:%s:", catID) }
func prodCatAllKey(catID string) string { return fmt.Sprintf("prod:cat:%s:all", catID) }
func prodAttrKey(catID string) string   { return fmt.Sprintf("prod:attr:%s", catID) }

// prodListKey builds a deterministic cache key from a ListProductsRequest by
// JSON-marshaling a canonical representation of all filter fields and hashing
// the result with MD5.
func prodListKey(req domain.ListProductsRequest) string {
	// Sort attribute filter values for determinism.
	sortedFilters := make(map[string][]string, len(req.AttributeFilters))
	for k, v := range req.AttributeFilters {
		sorted := make([]string, len(v))
		copy(sorted, v)
		sort.Strings(sorted)
		sortedFilters[k] = sorted
	}

	canonical := struct {
		Limit            int                 `json:"limit"`
		Cursor           string              `json:"cursor,omitempty"`
		SortBy           string              `json:"sort_by,omitempty"`
		SortDir          string              `json:"sort_dir,omitempty"`
		CategoryID       *string             `json:"category_id,omitempty"`
		Status           *domain.ProductStatus `json:"status,omitempty"`
		Search           string              `json:"search,omitempty"`
		MinPrice         *int64              `json:"min_price,omitempty"`
		MaxPrice         *int64              `json:"max_price,omitempty"`
		InStock          *bool               `json:"in_stock,omitempty"`
		LowStock         *bool               `json:"low_stock,omitempty"`
		Material         *string             `json:"material,omitempty"`
		Color            *string             `json:"color,omitempty"`
		Slug             string              `json:"slug,omitempty"`
		AttributeFilters map[string][]string `json:"attribute_filters,omitempty"`
	}{
		Limit:            req.Limit,
		Cursor:           req.Cursor,
		SortBy:           req.SortBy,
		SortDir:          req.SortDir,
		CategoryID:       req.CategoryID,
		Status:           req.Status,
		Search:           req.Search,
		MinPrice:         req.MinPrice,
		MaxPrice:         req.MaxPrice,
		InStock:          req.InStock,
		LowStock:         req.LowStock,
		Material:         req.Material,
		Color:            req.Color,
		Slug:             req.Slug,
		AttributeFilters: sortedFilters,
	}

	data, err := json.Marshal(canonical)
	if err != nil {
		return "" // empty key signals cache bypass
	}
	hash := md5.Sum(data)
	return prodListPrefix + hex.EncodeToString(hash[:])
}

// CachedProductRepository wraps a ProductRepository with an in-process cache.
type CachedProductRepository struct {
	inner domain.ProductRepository
	cache *cache.Cache
}

// NewCachedProductRepository returns a cache-decorated product repository.
func NewCachedProductRepository(inner domain.ProductRepository, c *cache.Cache) *CachedProductRepository {
	return &CachedProductRepository{inner: inner, cache: c}
}

func (r *CachedProductRepository) Create(ctx context.Context, product *domain.Product, inventory *domain.Inventory) error {
	err := r.inner.Create(ctx, product, inventory)
	if err == nil {
		r.invalidateForCategory(product.CategoryID)
	}
	return err
}

func (r *CachedProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	key := prodKey(id)
	if v, ok := r.cache.Get(key); ok {
		return v.(*domain.Product), nil
	}
	p, err := r.inner.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.cache.Set(key, p, prodItemTTL)
	return p, nil
}

func (r *CachedProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	key := prodSKUKey(sku)
	if v, ok := r.cache.Get(key); ok {
		return v.(*domain.Product), nil
	}
	p, err := r.inner.GetBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}
	r.cache.Set(key, p, prodItemTTL)
	return p, nil
}

func (r *CachedProductRepository) Update(ctx context.Context, product *domain.Product) error {
	err := r.inner.Update(ctx, product)
	if err == nil {
		r.cache.Delete(prodKey(product.ID))
		r.cache.Delete(prodSKUKey(product.SKU))
		r.invalidateForCategory(product.CategoryID)
	}
	return err
}

func (r *CachedProductRepository) Delete(ctx context.Context, id string) error {
	p, _ := r.inner.GetByID(ctx, id)
	err := r.inner.Delete(ctx, id)
	if err == nil {
		r.cache.Delete(prodKey(id))
		if p != nil {
			r.cache.Delete(prodSKUKey(p.SKU))
			r.invalidateForCategory(p.CategoryID)
		}
	}
	return err
}

func (r *CachedProductRepository) List(ctx context.Context, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	key := prodListKey(req)
	if key != "" {
		if v, ok := r.cache.Get(key); ok {
			return v.(*domain.ListProductsResponse), nil
		}
	}
	resp, err := r.inner.List(ctx, req)
	if err != nil {
		return nil, err
	}
	if key != "" {
		r.cache.Set(key, resp, prodListTTL)
	}
	return resp, nil
}

func (r *CachedProductRepository) BatchGetByIDs(ctx context.Context, ids []string) ([]*domain.Product, error) {
	results := make([]*domain.Product, 0, len(ids))
	var missIDs []string

	for _, id := range ids {
		if v, ok := r.cache.Get(prodKey(id)); ok {
			results = append(results, v.(*domain.Product))
		} else {
			missIDs = append(missIDs, id)
		}
	}

	if len(missIDs) == 0 {
		return results, nil
	}

	fetched, err := r.inner.BatchGetByIDs(ctx, missIDs)
	if err != nil {
		return nil, err
	}
	for _, p := range fetched {
		r.cache.Set(prodKey(p.ID), p, prodItemTTL)
		results = append(results, p)
	}
	return results, nil
}

func (r *CachedProductRepository) BatchUpdateSortOrder(ctx context.Context, products []*domain.Product) error {
	err := r.inner.BatchUpdateSortOrder(ctx, products)
	if err == nil && len(products) > 0 {
		for _, p := range products {
			r.cache.Delete(prodKey(p.ID))
		}
		r.invalidateForCategory(products[0].CategoryID)
	}
	return err
}

func (r *CachedProductRepository) GetByCategoryAll(ctx context.Context, categoryID string) ([]*domain.Product, error) {
	key := prodCatAllKey(categoryID)
	if v, ok := r.cache.Get(key); ok {
		return v.([]*domain.Product), nil
	}
	products, err := r.inner.GetByCategoryAll(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	r.cache.Set(key, products, prodItemTTL)
	return products, nil
}

func (r *CachedProductRepository) GetAttributeFilterOptions(ctx context.Context, categoryID string, attrNames []string) (map[string][]string, error) {
	key := prodAttrKey(categoryID)
	if v, ok := r.cache.Get(key); ok {
		return v.(map[string][]string), nil
	}
	vals, err := r.inner.GetAttributeFilterOptions(ctx, categoryID, attrNames)
	if err != nil {
		return nil, err
	}
	r.cache.Set(key, vals, prodAttrTTL)
	return vals, nil
}

func (r *CachedProductRepository) invalidateForCategory(categoryID string) {
	r.cache.DeletePrefix(prodCatPrefix(categoryID))
	r.cache.Delete(prodAttrKey(categoryID))
	r.cache.DeletePrefix(prodListPrefix)
}
