package postgres

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CentroidsRepository persists city centroids learned from CloudFront
// viewer headers on first sighting. Reads happen from the admin frontend
// via the Neon Data API; this Go-side repo only writes.
type CentroidsRepository struct {
	pool *pgxpool.Pool
	// seen dedups (city,country) sightings within a warm process so the hot
	// page-view path doesn't issue a PG round-trip on every event. The DB still
	// owns correctness via ON CONFLICT DO NOTHING; this is purely a round-trip
	// saver and is allowed to reset on cold start.
	seen sync.Map // key "city\x00country" -> struct{}
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
// Repeat sightings of a (city,country) already seen by this process short-
// circuit before touching PG. Callers should fire-and-forget — failures are
// logged/metered by the caller but never block the request path.
func (r *CentroidsRepository) Upsert(ctx context.Context, city, country string, lat, lng float64) error {
	key := city + "\x00" + country
	if _, ok := r.seen.Load(key); ok {
		return nil
	}
	const q = `
		INSERT INTO city_centroids (city, country, lat, lng)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (city, country) DO NOTHING
	`
	if _, err := r.pool.Exec(ctx, q, city, country, lat, lng); err != nil {
		return err
	}
	r.seen.Store(key, struct{}{})
	return nil
}
