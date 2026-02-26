package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/handloom/admin/internal/cache"
	"github.com/handloom/admin/internal/domain"
)

const (
	prodItemTTL = 1 * time.Hour
	prodAttrTTL = 1 * time.Hour
	prodPrefix  = "prod:"
)

func prodKey(id string) string         { return fmt.Sprintf("prod:%s", id) }
func prodCatPrefix(catID string) string { return fmt.Sprintf("prod:cat:%s:", catID) }
func prodAttrKey(catID string) string   { return fmt.Sprintf("prod:attr:%s", catID) }

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
	return r.inner.GetBySKU(ctx, sku)
}

func (r *CachedProductRepository) Update(ctx context.Context, product *domain.Product) error {
	err := r.inner.Update(ctx, product)
	if err == nil {
		r.cache.Delete(prodKey(product.ID))
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
			r.invalidateForCategory(p.CategoryID)
		}
	}
	return err
}

func (r *CachedProductRepository) List(ctx context.Context, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	return r.inner.List(ctx, req)
}

func (r *CachedProductRepository) BatchGetByIDs(ctx context.Context, ids []string) ([]*domain.Product, error) {
	return r.inner.BatchGetByIDs(ctx, ids)
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
	return r.inner.GetByCategoryAll(ctx, categoryID)
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
}
