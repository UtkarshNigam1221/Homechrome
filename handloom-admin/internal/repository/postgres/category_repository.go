package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/domain"
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
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		INSERT INTO categories (id, name, slug, description, image_url, status, product_count, created_at, updated_at, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		category.ID,
		category.Name,
		category.Slug,
		category.Description,
		category.ImageURL,
		string(category.Status),
		category.ProductCount,
		category.CreatedAt,
		category.UpdatedAt,
		category.CreatedBy,
		category.UpdatedBy,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperrors.Conflict("Category with this slug already exists")
		}
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

// GetByID retrieves a category by its ID, including all attributes and their options.
func (r *CategoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	// 1. Fetch the category row.
	cat := &domain.Category{}
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, description, image_url, status, product_count,
		       created_at, updated_at, created_by, updated_by
		FROM categories WHERE id = $1`, id,
	).Scan(
		&cat.ID, &cat.Name, &cat.Slug, &cat.Description, &cat.ImageURL,
		&status, &cat.ProductCount,
		&cat.CreatedAt, &cat.UpdatedAt, &cat.CreatedBy, &cat.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.New(apperrors.ErrCodeCategoryNotFound, "Category not found")
		}
		return nil, apperrors.Internal(err)
	}
	cat.Status = domain.CategoryStatus(status)

	// 2. Fetch attributes for this category.
	attrs, err := r.fetchAttributes(ctx, id)
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
	defer tx.Rollback(ctx) //nolint:errcheck

	cmdTag, err := tx.Exec(ctx, `
		UPDATE categories
		SET name = $2, slug = $3, description = $4, image_url = $5,
		    status = $6, updated_at = $7, updated_by = $8
		WHERE id = $1`,
		category.ID,
		category.Name,
		category.Slug,
		category.Description,
		category.ImageURL,
		string(category.Status),
		category.UpdatedAt,
		category.UpdatedBy,
	)
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

	// Build the dynamic query.
	query := `SELECT id, name, slug, description, image_url, status, product_count,
	                 created_at, updated_at, created_by, updated_by
	          FROM categories`

	var conditions []string
	var args []interface{}
	argIdx := 1

	if req.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++
	}
	if req.Search != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIdx))
		args = append(args, "%"+req.Search+"%")
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, cond := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += cond
		}
	}

	query += " ORDER BY created_at DESC"

	// Fetch limit+1 to determine HasMore.
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit+1, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	defer rows.Close()

	var categories []*domain.Category
	for rows.Next() {
		cat := &domain.Category{}
		var status string
		if err := rows.Scan(
			&cat.ID, &cat.Name, &cat.Slug, &cat.Description, &cat.ImageURL,
			&status, &cat.ProductCount,
			&cat.CreatedAt, &cat.UpdatedAt, &cat.CreatedBy, &cat.UpdatedBy,
		); err != nil {
			return nil, apperrors.Internal(err)
		}
		cat.Status = domain.CategoryStatus(status)
		categories = append(categories, cat)
	}
	if err := rows.Err(); err != nil {
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
	cmdTag, err := r.pool.Exec(ctx, `
		UPDATE categories
		SET product_count = product_count + $2, updated_at = NOW()
		WHERE id = $1`,
		id, delta,
	)
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
		_, err := tx.Exec(ctx, `
			INSERT INTO category_attributes (id, category_id, name, label, type, required, searchable, display_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			attrID,
			categoryID,
			attr.Name,
			attr.Label,
			string(attr.Type),
			attr.Required,
			attr.Searchable,
			attr.DisplayOrder,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return apperrors.Conflict(fmt.Sprintf("Duplicate attribute name %q for category", attr.Name))
			}
			return apperrors.Internal(err)
		}

		for sortOrder, opt := range attr.Options {
			optID := uuid.New().String()
			_, err := tx.Exec(ctx, `
				INSERT INTO category_attribute_options (id, attribute_id, value, label, sort_order)
				VALUES ($1, $2, $3, $4, $5)`,
				optID,
				attrID,
				opt.Value,
				opt.Label,
				sortOrder,
			)
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
