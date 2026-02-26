package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres/querybuilder"
	apperrors "github.com/handloom/admin/pkg/errors"
)

// CategoryRepository implements domain.CategoryRepository using PostgreSQL.
type CategoryRepository struct {
	pool *pgxpool.Pool
}

// NewCategoryRepository creates a new CategoryRepository backed by PostgreSQL.
func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

// Create inserts a new category along with its attributes and attribute options
// in a single transaction.
func (r *CategoryRepository) Create(ctx context.Context, category *domain.Category) error {
	now := time.Now().UTC()
	category.CreatedAt = now
	category.UpdatedAt = now

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperrors.Internal(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qb := querybuilder.Insert("categories").
		Set(ColID, category.ID).
		Set(ColName, category.Name).
		Set(ColSlug, category.Slug).
		Set(ColDescription, category.Description).
		Set(ColImageURL, category.ImageURL).
		Set(ColStatus, string(category.Status)).
		Set(ColProductCount, category.ProductCount).
		Set(ColCreatedAt, category.CreatedAt).
		Set(ColUpdatedAt, category.UpdatedAt).
		Set(ColCreatedBy, category.CreatedBy).
		Set(ColUpdatedBy, category.UpdatedBy)

	query, args := qb.Build()
	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperrors.Conflict("Category with this slug already exists")
		}
		return apperrors.Internal(err)
	}

	if err = insertAttributes(ctx, tx, category.ID, category.OwnAttributes); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return apperrors.Internal(err)
	}

	return nil
}

// GetByID retrieves a category by its ID, including all attributes and their options.
func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	qb := querybuilder.Select(categoryColumns...).From("categories").Where(ColID, id)
	query, args := qb.Build()

	cat := &domain.Category{}
	if err := pgxscan.Get(ctx, r.pool, cat, query, args...); err != nil {
		if pgxscan.NotFound(err) {
			return nil, apperrors.New(apperrors.ErrCodeCategoryNotFound, "Category not found")
		}
		return nil, apperrors.Internal(err)
	}

	attrs, err := r.fetchAttributes(ctx, cat.ID)
	if err != nil {
		return nil, err
	}
	cat.OwnAttributes = attrs

	return cat, nil
}

// Update updates an existing category's scalar fields and replaces its attributes
// (deleting old ones via CASCADE and re-inserting) inside a transaction.
func (r *CategoryRepository) Update(ctx context.Context, category *domain.Category) error {
	category.UpdatedAt = time.Now().UTC()

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperrors.Internal(err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qb := querybuilder.Update("categories").
		Set(ColName, category.Name).
		Set(ColSlug, category.Slug).
		Set(ColDescription, category.Description).
		Set(ColImageURL, category.ImageURL).
		Set(ColStatus, string(category.Status)).
		Set(ColUpdatedAt, category.UpdatedAt).
		Set(ColUpdatedBy, category.UpdatedBy).
		Where(ColID, category.ID)

	query, args := qb.Build()
	cmdTag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperrors.Conflict("Category with this slug already exists")
		}
		return apperrors.Internal(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return apperrors.New(apperrors.ErrCodeCategoryNotFound, "Category not found")
	}

	// Delete existing attributes (CASCADE deletes options too), then re-insert.
	_, err = tx.Exec(ctx, `DELETE FROM category_attributes WHERE category_id = $1`, category.ID)
	if err != nil {
		return apperrors.Internal(err)
	}

	if err := insertAttributes(ctx, tx, category.ID, category.OwnAttributes); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.Internal(err)
	}

	return nil
}

// Delete removes a category by ID. The CASCADE foreign key on category_attributes
// and category_attribute_options ensures children are cleaned up automatically.
func (r *CategoryRepository) Delete(ctx context.Context, id string) error {
	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return apperrors.Internal(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return apperrors.New(apperrors.ErrCodeCategoryNotFound, "Category not found")
	}
	return nil
}

// List retrieves categories with optional status filter and name search,
// ordered by created_at DESC with cursor-based pagination.
func (r *CategoryRepository) List(ctx context.Context, req domain.ListCategoriesRequest) (*domain.ListCategoriesResponse, error) {
	limit, offset := pageParams(req.PaginationRequest)

	qb := querybuilder.Select(categoryColumns...).
		From("categories").
		WithFilter(req.Status != nil, ColStatus, string(deref(req.Status))).
		WithLike(req.Search != "", ColName, "%"+req.Search+"%").
		OrderBy(ColCreatedAt + " DESC").
		Limit(limit + 1).
		Offset(offset)

	query, args := qb.Build()

	var categories []*domain.Category
	if err := pgxscan.Select(ctx, r.pool, &categories, query, args...); err != nil {
		return nil, apperrors.Internal(err)
	}

	pagination := buildPaginationResponse(limit, offset, len(categories))

	// Trim the extra item used for HasMore detection.
	if len(categories) > limit {
		categories = categories[:limit]
	}

	return &domain.ListCategoriesResponse{
		Categories: categories,
		Pagination: pagination,
	}, nil
}

// IncrementProductCount atomically increments (or decrements) the product_count
// for a category and updates the updated_at timestamp.
func (r *CategoryRepository) IncrementProductCount(ctx context.Context, id string, delta int) error {
	qb := querybuilder.Update("categories").
		SetRaw(ColProductCount, ColProductCount+" + %s", delta).
		SetRaw(ColUpdatedAt, "NOW()").
		Where(ColID, id)

	query, args := qb.Build()
	cmdTag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return apperrors.Internal(err)
	}
	if cmdTag.RowsAffected() == 0 {
		return apperrors.New(apperrors.ErrCodeCategoryNotFound, "Category not found")
	}
	return nil
}

// ---------- internal helpers ----------

// insertAttributes batch-inserts category attributes and their options into the
// provided transaction.
func insertAttributes(ctx context.Context, tx pgx.Tx, categoryID string, attrs []domain.CategoryAttribute) error {
	for _, attr := range attrs {
		attrID := uuid.New().String()
		attrQB := querybuilder.Insert("category_attributes").
			Set(ColID, attrID).
			Set(ColCategoryID, categoryID).
			Set(ColName, attr.Name).
			Set(ColLabel, attr.Label).
			Set(ColType, string(attr.Type)).
			Set(ColRequired, attr.Required).
			Set(ColSearchable, attr.Searchable).
			Set(ColDisplayOrder, attr.DisplayOrder)

		attrSQL, attrArgs := attrQB.Build()
		_, err := tx.Exec(ctx, attrSQL, attrArgs...)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return apperrors.Conflict(fmt.Sprintf("Duplicate attribute name %q for category", attr.Name))
			}
			return apperrors.Internal(err)
		}

		for sortOrder, opt := range attr.Options {
			optID := uuid.New().String()
			optQB := querybuilder.Insert("category_attribute_options").
				Set(ColID, optID).
				Set(ColAttributeID, attrID).
				Set(ColValue, opt.Value).
				Set(ColLabel, opt.Label).
				Set(ColSortOrder, sortOrder)

			optSQL, optArgs := optQB.Build()
			_, err := tx.Exec(ctx, optSQL, optArgs...)
			if err != nil {
				var pgErr *pgconn.PgError
				if errors.As(err, &pgErr) && pgErr.Code == "23505" {
					return apperrors.Conflict(fmt.Sprintf("Duplicate option value %q for attribute %q", opt.Value, attr.Name))
				}
				return apperrors.Internal(err)
			}
		}
	}
	return nil
}

// attrRow is an intermediate struct for scanning joined attribute+option rows.
type attrRow struct {
	// attribute fields
	AttrName         string
	AttrLabel        string
	AttrType         string
	AttrRequired     bool
	AttrSearchable   bool
	AttrDisplayOrder int
	// option fields (nullable)
	OptValue *string
	OptLabel *string
}

// fetchAttributes loads category attributes and their options for a given
// category ID and assembles them into domain types.
func (r *CategoryRepository) fetchAttributes(ctx context.Context, categoryID string) ([]domain.CategoryAttribute, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.name, a.label, a.type, a.required, a.searchable, a.display_order,
		       o.value, o.label
		FROM category_attributes a
		LEFT JOIN category_attribute_options o ON o.attribute_id = a.id
		WHERE a.category_id = $1
		ORDER BY a.display_order, a.name, o.sort_order`,
		categoryID,
	)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	defer rows.Close()

	// Use an ordered map to preserve attribute ordering.
	type attrEntry struct {
		attr    domain.CategoryAttribute
		options []domain.AttributeOption
	}
	attrMap := make(map[string]*attrEntry)
	var attrOrder []string

	for rows.Next() {
		var ar attrRow
		if err := rows.Scan(
			&ar.AttrName, &ar.AttrLabel, &ar.AttrType,
			&ar.AttrRequired, &ar.AttrSearchable, &ar.AttrDisplayOrder,
			&ar.OptValue, &ar.OptLabel,
		); err != nil {
			return nil, apperrors.Internal(err)
		}

		entry, exists := attrMap[ar.AttrName]
		if !exists {
			entry = &attrEntry{
				attr: domain.CategoryAttribute{
					Name:         ar.AttrName,
					Label:        ar.AttrLabel,
					Type:         domain.AttributeType(ar.AttrType),
					Required:     ar.AttrRequired,
					Searchable:   ar.AttrSearchable,
					DisplayOrder: ar.AttrDisplayOrder,
				},
			}
			attrMap[ar.AttrName] = entry
			attrOrder = append(attrOrder, ar.AttrName)
		}

		if ar.OptValue != nil {
			optLabel := ""
			if ar.OptLabel != nil {
				optLabel = *ar.OptLabel
			}
			entry.options = append(entry.options, domain.AttributeOption{
				Value: *ar.OptValue,
				Label: optLabel,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal(err)
	}

	attrs := make([]domain.CategoryAttribute, 0, len(attrOrder))
	for _, name := range attrOrder {
		entry := attrMap[name]
		entry.attr.Options = entry.options
		attrs = append(attrs, entry.attr)
	}

	return attrs, nil
}

// Ensure interface compliance at compile time.
var _ domain.CategoryRepository = (*CategoryRepository)(nil)
