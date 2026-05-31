package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CentroidsRepository persists city centroids learned from CloudFront
// viewer headers on first sighting. Reads happen from the admin frontend
// via the Neon Data API; this Go-side repo only writes.
type CentroidsRepository struct {
	pool *pgxpool.Pool
}

// NewCentroidsRepository creates a new CentroidsRepository.
func NewCentroidsRepository(pool *pgxpool.Pool) *CentroidsRepository {
	return &CentroidsRepository{pool: pool}
}

// Upsert inserts a (city, country, lat, lng) row, no-oping on conflict.
// The first sighting wins: subsequent emits from the same city keep the
// original centroid even if CloudFront geo data drifts slightly. This is
// intentional — dashboard markers stay stable across rebuilds.
//
// Callers should fire-and-forget — failures are logged but never block
// the calling request path.
func (r *CentroidsRepository) Upsert(ctx context.Context, city, country string, lat, lng float64) error {
	const q = `
		INSERT INTO city_centroids (city, country, lat, lng)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (city, country) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, q, city, country, lat, lng)
	return err
}
