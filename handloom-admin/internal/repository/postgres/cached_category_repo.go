package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/handloom/admin/internal/cache"
	"github.com/handloom/admin/internal/domain"
)

const (
	catListKey   = "cat:list"
	catListTTL   = 1 * time.Hour
	catItemTTL   = 1 * time.Hour
	catKeyPrefix = "cat:"
)

func catKey(id string) string { return fmt.Sprintf("cat:%s", id) }

// CachedCategoryRepository wraps a CategoryRepository with an in-process cache.
type CachedCategoryRepository struct {
	inner domain.CategoryRepository
	cache *cache.Cache
}

// NewCachedCategoryRepository returns a cache-decorated category repository.
func NewCachedCategoryRepository(inner domain.CategoryRepository, c *cache.Cache) *CachedCategoryRepository {
	return &CachedCategoryRepository{inner: inner, cache: c}
}

func (r *CachedCategoryRepository) Create(ctx context.Context, category *domain.Category) error {
	err := r.inner.Create(ctx, category)
	if err == nil {
		r.cache.Delete(catListKey)
	}
	return err
}

func (r *CachedCategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	key := catKey(id)
	if v, ok := r.cache.Get(key); ok {
		if cat, ok := v.(*domain.Category); ok {
			return cat, nil
		}
	}
	cat, err := r.inner.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	r.cache.Set(key, cat, catItemTTL)
	return cat, nil
}

func (r *CachedCategoryRepository) Update(ctx context.Context, category *domain.Category) error {
	err := r.inner.Update(ctx, category)
	if err == nil {
		r.cache.Delete(catKey(category.ID))
		r.cache.Delete(catListKey)
	}
	return err
}

func (r *CachedCategoryRepository) Delete(ctx context.Context, id string) error {
	err := r.inner.Delete(ctx, id)
	if err == nil {
		r.cache.Delete(catKey(id))
		r.cache.Delete(catListKey)
	}
	return err
}

func (r *CachedCategoryRepository) List(ctx context.Context, req domain.ListCategoriesRequest) (*domain.ListCategoriesResponse, error) {
	// Only cache unfiltered first-page requests
	if req.Status == nil && req.Search == "" && req.Cursor == "" {
		if v, ok := r.cache.Get(catListKey); ok {
			if resp, ok := v.(*domain.ListCategoriesResponse); ok {
				return resp, nil
			}
		}
	}
	resp, err := r.inner.List(ctx, req)
	if err != nil {
		return nil, err
	}
	if req.Status == nil && req.Search == "" && req.Cursor == "" {
		r.cache.Set(catListKey, resp, catListTTL)
	}
	return resp, nil
}

func (r *CachedCategoryRepository) IncrementProductCount(ctx context.Context, id string, delta int) error {
	err := r.inner.IncrementProductCount(ctx, id, delta)
	if err == nil {
		r.cache.Delete(catKey(id))
		r.cache.Delete(catListKey)
	}
	return err
}
