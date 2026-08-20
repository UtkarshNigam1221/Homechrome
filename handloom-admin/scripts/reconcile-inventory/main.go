// Reports stock reserved against orders that never dispatched or cancelled.
// Exits 1 when any is found, so a scheduled run failing is the alert.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/repository/postgres"
)

func main() {
	minAge := flag.Duration("min-age", 24*time.Hour, "ignore reservations younger than this")
	limit := flag.Int("limit", 500, "maximum rows to report")
	flag.Parse()

	ctx := context.Background()
	stranded := 0
	checked := 0
	failed := false

	// Each environment is named by the variable holding its DSN, so a missing
	// secret is reported rather than silently skipping that environment.
	for _, env := range []struct{ name, dsnVar string }{
		{"dev", "DEV_DSN"},
		{"prod", "PROD_DSN"},
	} {
		dsn := os.Getenv(env.dsnVar)
		if dsn == "" {
			fmt.Printf("::warning::%s: %s is empty, skipping\n", env.name, env.dsnVar)
			continue
		}
		checked++

		units, err := report(ctx, env.name, dsn, *minAge, *limit)
		if err != nil {
			// A read that failed is not a clean environment.
			fmt.Printf("::error::%s: could not read the ledger: %v\n", env.name, err)
			failed = true
			continue
		}
		stranded += units
	}

	if checked == 0 {
		fmt.Println("::error::no environment could be checked — both DSNs were empty")
		os.Exit(1)
	}
	if failed || stranded > 0 {
		os.Exit(1)
	}
}

func report(ctx context.Context, name, dsn string, minAge time.Duration, limit int) (int, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return 0, fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	orphans, err := postgres.NewInventoryRepository(pool).FindOrphanReservations(ctx, minAge, limit)
	if err != nil {
		return 0, err
	}
	if len(orphans) == 0 {
		fmt.Printf("%s: clean — nothing unsettled beyond %s\n", name, minAge)
		return 0, nil
	}

	units := 0
	fmt.Printf("::group::%s: %d order(s) holding stock against nothing\n", name, len(orphans))
	for _, o := range orphans {
		units += o.Quantity
		fmt.Printf("  order %-28s %-16s %-30s %3d unit(s)  since %s\n",
			o.OrderID, o.SKU, o.ProductName, o.Quantity, o.ReservedAt.Format(time.DateOnly))
	}
	fmt.Println("::endgroup::")

	// Truncation means the drift is systemic; say so rather than leave the
	// reader to notice a suspiciously round number.
	if len(orphans) == limit {
		fmt.Printf("::warning::%s: hit the %d-row limit, there may be more\n", name, limit)
	}
	fmt.Printf("::error::%s: %d unit(s) reserved against %d order(s) that never dispatched or cancelled\n",
		name, units, len(orphans))
	return units, nil
}
