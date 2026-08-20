// Command promote-catalog copies catalog data (categories, products, media)
// from dev to prod. Additive-only: everything upserts by id and nothing
// prod-only is ever deleted; a promoted product's attribute values and images
// are replaced as a set; asset URLs are rewritten to the prod CDN; only S3
// objects referenced by promoted rows are copied. DynamoDB data and
// inventory_transactions are never touched.
//
// Stock does not promote. A promoted product arrives with an inventory row at
// zero and is stocked up by hand in prod. Dev quantities are test artifacts —
// they are whatever the last e2e run or manual poke left behind — and carrying
// them over would make prod's opening balance a number nobody chose. Stocking
// up through the admin API also writes a proper ADD ledger row with a real
// actor and reason, so prod's ledger reconciles from its first movement;
// seeding a quantity directly would leave stock with no entry behind it.
// Existing prod stock is never touched.
//
// Usage: go run ./scripts/promote-catalog [--products active|all|id1,id2,...]
// [--skip-s3] [--yes]
//
// Env: DEV_DSN/PROD_DSN (default: POSTGRES_DSN from .env.{dev,prod}),
// DEV_ASSET_HOST/PROD_ASSET_HOST (default: CloudFormation exports), AWS_REGION.
package main

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	devBucket  = "handloom-assets-dev"
	prodBucket = "handloom-assets-prod"

	colURL      = "url" // product_images URL column
	whereActive = "status = 'ACTIVE'"
)

var idListRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// column is one insertable column. Values travel dev->prod as text and are
// cast back to typ on insert, so pgx never needs pgvector/array type support.
type column struct {
	name string
	typ  string
}

func main() {
	log.SetFlags(0)

	products := flag.String("products", "active", `which products to promote: "active", "all", or comma-separated ids`)
	skipS3 := flag.Bool("skip-s3", false, "skip S3 media sync")
	yes := flag.Bool("yes", false, "skip confirmation prompt (CI)")
	flag.Parse()

	ctx := context.Background()
	region := cmp.Or(os.Getenv("AWS_REGION"), "ap-south-1")
	if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(region) {
		log.Fatal("invalid AWS_REGION env var")
	}

	devDSN := dsnOrEnvFile("DEV_DSN", ".env.dev")
	prodDSN := dsnOrEnvFile("PROD_DSN", ".env.prod")

	devHost := assetHost(ctx, "DEV_ASSET_HOST", "handloom-assets-cdn-dev", region)
	prodHost := assetHost(ctx, "PROD_ASSET_HOST", "handloom-assets-cdn-prod", region)
	// Rewrites applied to every URL column. Legacy pre-CDN direct-S3 URLs are
	// rewritten to the prod CDN too.
	urlRewrites := [][2]string{
		{"https://" + devHost + "/", "https://" + prodHost + "/"},
		{fmt.Sprintf("https://%s.s3.%s.amazonaws.com/", devBucket, region), "https://" + prodHost + "/"},
	}

	productWhere, productArgs, wantIDs := productFilter(*products)
	childWhere := "product_id IN (SELECT id FROM products WHERE " + productWhere + ")"

	dev, err := pgx.Connect(ctx, devDSN)
	if err != nil {
		log.Fatalf("connect dev: %v", err)
	}
	defer func() { _ = dev.Close(ctx) }()
	prod, err := pgx.Connect(ctx, prodDSN)
	if err != nil {
		log.Fatalf("connect prod: %v", err)
	}
	defer func() { _ = prod.Close(ctx) }()

	cols := loadColumns(ctx, dev, prod)

	// Fetch everything from dev up front so the plan shows exact counts.
	fetch := func(table, where string, args ...any) [][]any {
		rows, fetchErr := fetchTextRows(ctx, dev, table, cols[table], where, args...)
		if fetchErr != nil {
			log.Fatalf("fetch %s from dev: %v", table, fetchErr)
		}
		return rows
	}
	categories := fetch("categories", "TRUE")
	categoryAttrs := fetch("category_attributes", "TRUE")
	categoryOpts := fetch("category_attribute_options", "TRUE")
	prods := fetch("products", productWhere, productArgs...)
	attrValues := fetch("product_attribute_values", childWhere, productArgs...)
	images := fetch("product_images", childWhere, productArgs...)
	inventory := fetch("inventory", childWhere, productArgs...)

	verifySelection(*products, prods, cols["products"], wantIDs)

	// A prod-only product already owning a promoted sku/slug would abort the
	// transaction on the unique index — surface it up front, readably.
	checkUniqueOwnership(ctx, prod, prods, cols["products"])

	// Only media referenced by promoted rows is copied — a full-prefix sync
	// would publish dev-only media (drafts, test uploads) through the prod CDN.
	var mediaKeys []string
	if !*skipS3 {
		mediaKeys = assetKeys(devHost, region,
			urlValues(categories, cols["categories"], "image_url"),
			urlValues(prods, cols["products"], "video_url", "video_poster_url"),
			urlValues(images, cols["product_images"], colURL))
	}

	inventoryMode := "seed missing rows at zero — stock up in prod by hand; existing prod stock untouched"
	mediaMode := fmt.Sprintf("%d referenced objects -> s3://%s/assets/ (additive)", len(mediaKeys), prodBucket)
	if *skipS3 {
		mediaMode = "skipped"
	}
	fmt.Printf(`
Promotion plan (dev -> prod):
  categories:  %d (+ %d attributes, %d options)
  products:    %d (--products %s) (+ %d attribute values, %d images)
  inventory:   %s
  media:       %s
  url rewrite: https://%s/ -> https://%s/

`, len(categories), len(categoryAttrs), len(categoryOpts),
		len(prods), *products, len(attrValues), len(images),
		inventoryMode, mediaMode, devHost, prodHost)

	if !*yes && !confirm("Write to PROD? [y/N] ") {
		log.Fatal("aborted")
	}

	// Media first, so no DB row ever points at a missing object.
	if !*skipS3 && len(mediaKeys) > 0 {
		syncMedia(ctx, region, mediaKeys)
	}

	rewriteURLs(categories, cols["categories"], []string{"image_url"}, urlRewrites)
	rewriteURLs(prods, cols["products"], []string{"video_url", "video_poster_url"}, urlRewrites)
	rewriteURLs(images, cols["product_images"], []string{colURL}, urlRewrites)

	// Stock and reservations both belong to prod. The row is seeded at zero so
	// the product exists and can be stocked up through the admin API, which
	// writes the ADD ledger entry that a directly-seeded quantity would not.
	// Thresholds carry over: those are catalog configuration, not stock.
	iQty := colIndex(cols["inventory"], "quantity")
	iRes := colIndex(cols["inventory"], "reserved_qty")
	iAvail := colIndex(cols["inventory"], "available_qty")
	for _, row := range inventory {
		row[iQty] = "0"
		row[iRes] = "0"
		row[iAvail] = "0"
	}
	// DO NOTHING, so a re-promotion never zeroes stock somebody has since added.
	inventoryConflict := "ON CONFLICT (product_id) DO NOTHING"

	// Single transaction: either the whole catalog lands or none of it.
	tx, err := prod.Begin(ctx)
	if err != nil {
		log.Fatalf("begin prod tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	step := func(name string, n int, err error) {
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				log.Fatalf("%s: unique constraint %s violated (%s) — a promoted row collides with an existing one; resolve the duplicate and retry",
					name, pgErr.ConstraintName, pgErr.Detail)
			}
			log.Fatalf("%s: %v", name, err)
		}
		log.Printf("  %-28s %d rows", name, n)
	}

	// Categories: upsert by id, keeping prod's creation stamps and its own
	// product_count (recomputed below — prod may hold products dev never had).
	n, err := insertRows(ctx, tx, "categories", cols["categories"], categories,
		"ON CONFLICT (id) DO UPDATE SET "+updateSetClause(cols["categories"], "id", "product_count", "created_at", "created_by"))
	step("categories", n, err)

	// Category attribute definitions + options: upsert by id, no deletes —
	// definitions added directly on prod survive (additive-only guarantee).
	n, err = insertRows(ctx, tx, "category_attributes", cols["category_attributes"], categoryAttrs,
		"ON CONFLICT (id) DO UPDATE SET "+updateSetClause(cols["category_attributes"], "id"))
	step("category_attributes", n, err)
	n, err = insertRows(ctx, tx, "category_attribute_options", cols["category_attribute_options"], categoryOpts,
		"ON CONFLICT (id) DO UPDATE SET "+updateSetClause(cols["category_attribute_options"], "id"))
	step("category_attribute_options", n, err)

	// Products: upsert by id. search_vector regenerates; embeddings copy over.
	n, err = insertRows(ctx, tx, "products", cols["products"], prods,
		"ON CONFLICT (id) DO UPDATE SET "+updateSetClause(cols["products"], "id", "created_at", "created_by"))
	step("products", n, err)

	// Attribute values + images: replace as a set, scoped to promoted products.
	productIDs := idsOf(prods, cols["products"])
	err = deleteScoped(ctx, tx, "product_attribute_values", "product_id", productIDs)
	if err == nil {
		n, err = insertRows(ctx, tx, "product_attribute_values", cols["product_attribute_values"], attrValues, "")
	}
	step("product_attribute_values", n, err)
	err = deleteScoped(ctx, tx, "product_images", "product_id", productIDs)
	if err == nil {
		n, err = insertRows(ctx, tx, "product_images", cols["product_images"], images, "")
	}
	step("product_images", n, err)

	n, err = insertRows(ctx, tx, "inventory", cols["inventory"], inventory, inventoryConflict)
	step("inventory", n, err)

	// Denormalized counter must reflect prod's actual rows, not dev's.
	if _, cntErr := tx.Exec(ctx, `UPDATE categories c SET product_count = (SELECT count(*) FROM products p WHERE p.category_id = c.id)`); cntErr != nil {
		log.Fatalf("recompute product_count: %v", cntErr)
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		log.Fatalf("commit: %v", commitErr)
	}

	var nCat, nProd, nImg, nInv int
	err = prod.QueryRow(ctx, `SELECT (SELECT count(*) FROM categories), (SELECT count(*) FROM products),
	                                 (SELECT count(*) FROM product_images), (SELECT count(*) FROM inventory)`).
		Scan(&nCat, &nProd, &nImg, &nInv)
	if err != nil {
		log.Fatalf("prod totals: %v", err)
	}
	fmt.Printf("\nDone. Prod now has %d categories, %d products, %d images, %d inventory rows.\n", nCat, nProd, nImg, nInv)
	fmt.Println("Note: catalog responses carry Cache-Control max-age=3600 (cached by CloudFront and browsers) — run a CloudFront invalidation on the prod distribution for instant visibility, or wait up to 1h.")
}

// productFilter maps the --products flag to a WHERE clause + args. For an id
// list it also returns the requested ids so the caller can verify coverage.
func productFilter(products string) (string, []any, []string) {
	switch strings.ToLower(products) {
	case "active":
		return whereActive, nil, nil
	case "all":
		return "TRUE", nil, nil
	default:
		ids := strings.Split(products, ",")
		for _, id := range ids {
			if !idListRe.MatchString(id) {
				log.Fatalf("--products must be \"active\", \"all\", or a comma-separated id list (bad id: %q)", id)
			}
		}
		return "id = ANY($1)", []any{ids}, ids
	}
}

// loadColumns reads every table's insertable columns from both sides and
// fails on drift: prod needs a deploy (which runs the migrator) first.
func loadColumns(ctx context.Context, dev, prod *pgx.Conn) map[string][]column {
	tables := []string{
		"categories", "category_attributes", "category_attribute_options",
		"products", "product_attribute_values", "product_images", "inventory",
	}
	cols := map[string][]column{}
	for _, t := range tables {
		devCols, err := columnsOf(ctx, dev, t)
		if err != nil {
			log.Fatalf("read dev schema for %s: %v", t, err)
		}
		prodCols, err := columnsOf(ctx, prod, t)
		if err != nil {
			log.Fatalf("read prod schema for %s: %v", t, err)
		}
		if fmt.Sprint(devCols) != fmt.Sprint(prodCols) {
			log.Fatalf("schema drift on %q:\n  dev:  %v\n  prod: %v\ndeploy the backend to prod first (runs the migrator), then retry", t, devCols, prodCols)
		}
		cols[t] = devCols
	}
	return cols
}

// verifySelection refuses selections that match nothing (an operator mistake,
// e.g. "Active" parsed as an id) or id lists with ids dev doesn't have.
func verifySelection(products string, prods [][]any, cols []column, wantIDs []string) {
	if len(prods) == 0 {
		log.Fatalf("--products %q matched no dev products — nothing to promote", products)
	}
	if wantIDs == nil {
		return
	}
	got := make(map[string]bool, len(prods))
	for _, id := range idsOf(prods, cols) {
		got[id] = true
	}
	var missing []string
	for _, id := range wantIDs {
		if !got[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		log.Fatalf("--products ids not found in dev: %s", strings.Join(missing, ", "))
	}
}

// syncMedia copies exactly the referenced objects dev -> prod (additive).
func syncMedia(ctx context.Context, region string, keys []string) {
	log.Printf("Syncing %d media objects...", len(keys))
	// ponytail: one sync call with N --include flags; chunk the calls if a
	// catalog ever references thousands of objects.
	args := make([]string, 0, 8+2*len(keys))
	args = append(args, "s3", "sync",
		"s3://"+devBucket+"/assets/", "s3://"+prodBucket+"/assets/",
		"--region", region, "--exclude", "*")
	for _, k := range keys {
		args = append(args, "--include", k)
	}
	sync := exec.CommandContext(ctx, "aws", args...) //nolint:gosec // fixed command; keys come from our own DB rows
	sync.Stdout, sync.Stderr = os.Stdout, os.Stderr
	if err := sync.Run(); err != nil {
		log.Fatalf("s3 sync: %v", err)
	}
}

// checkUniqueOwnership fails fast when a prod-only product already owns a
// promoted sku or slug — the upsert-by-id would otherwise abort mid-run on
// the unique index with a raw constraint error.
func checkUniqueOwnership(ctx context.Context, prod *pgx.Conn, rows [][]any, cols []column) {
	iID, iSKU, iSlug := colIndex(cols, "id"), colIndex(cols, "sku"), colIndex(cols, "slug")
	devByID := make(map[string]bool, len(rows))
	skus := make([]string, 0, len(rows))
	slugs := make([]string, 0, len(rows))
	for _, r := range rows {
		id, _ := r[iID].(string)
		sku, _ := r[iSKU].(string)
		slug, _ := r[iSlug].(string)
		devByID[id] = true
		skus = append(skus, sku)
		slugs = append(slugs, slug)
	}
	res, err := prod.Query(ctx, `SELECT id, sku, slug FROM products WHERE sku = ANY($1) OR slug = ANY($2)`, skus, slugs)
	if err != nil {
		log.Fatalf("preflight unique check: %v", err)
	}
	defer res.Close()
	var conflicts []string
	for res.Next() {
		var id, sku, slug string
		if scanErr := res.Scan(&id, &sku, &slug); scanErr != nil {
			log.Fatalf("preflight unique check: %v", scanErr)
		}
		if !devByID[id] {
			conflicts = append(conflicts, fmt.Sprintf("prod product %s owns sku=%q slug=%q", id, sku, slug))
		}
	}
	if res.Err() != nil {
		log.Fatalf("preflight unique check: %v", res.Err())
	}
	if len(conflicts) > 0 {
		log.Fatalf("promotion would collide with prod-only products on unique sku/slug:\n  %s\nrename those on prod (or exclude the dev products) and retry",
			strings.Join(conflicts, "\n  "))
	}
}

// urlValues collects non-empty values of the named URL columns.
func urlValues(rows [][]any, cols []column, urlCols ...string) []string {
	var out []string
	for _, c := range urlCols {
		i := colIndex(cols, c)
		for _, row := range rows {
			if s, ok := row[i].(string); ok && s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

// assetKeys converts dev asset URLs into object keys relative to the assets/
// prefix, deduped, for use as s3 sync --include patterns.
func assetKeys(devHost, region string, urlGroups ...[]string) []string {
	prefixes := []string{
		"https://" + devHost + "/assets/",
		fmt.Sprintf("https://%s.s3.%s.amazonaws.com/assets/", devBucket, region),
	}
	seen := map[string]bool{}
	var keys []string
	for _, group := range urlGroups {
		for _, u := range group {
			for _, p := range prefixes {
				if k, ok := strings.CutPrefix(u, p); ok && k != "" && !seen[k] {
					seen[k] = true
					keys = append(keys, k)
				}
			}
		}
	}
	return keys
}

// columnsOf returns the insertable columns of a table, excluding generated
// columns (products.search_vector can't be inserted).
func columnsOf(ctx context.Context, conn *pgx.Conn, table string) ([]column, error) {
	rows, err := conn.Query(ctx, `
		SELECT a.attname, format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relname = $1
		  AND a.attnum > 0 AND NOT a.attisdropped AND a.attgenerated = ''
		ORDER BY a.attnum`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []column
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.name, &c.typ); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("table %q not found", table)
	}
	return cols, rows.Err()
}

// fetchTextRows selects every column cast to text, so values of any type
// (vector, text[], numeric, timestamptz) round-trip losslessly as strings.
func fetchTextRows(ctx context.Context, conn *pgx.Conn, table string, cols []column, where string, args ...any) ([][]any, error) {
	sel := make([]string, len(cols))
	for i, c := range cols {
		sel[i] = ident(c.name) + "::text"
	}
	rows, err := conn.Query(ctx,
		fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(sel, ", "), ident(table), where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		out = append(out, vals)
	}
	return out, rows.Err()
}

// insertRows inserts rows one batch, casting each text value back to the
// column's real type. onConflict is appended verbatim ("" = plain insert).
func insertRows(ctx context.Context, tx pgx.Tx, table string, cols []column, rows [][]any, onConflict string) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	names := make([]string, len(cols))
	placeholders := make([]string, len(cols))
	for i, c := range cols {
		names[i] = ident(c.name)
		placeholders[i] = fmt.Sprintf("$%d::%s", i+1, c.typ)
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) %s",
		ident(table), strings.Join(names, ", "), strings.Join(placeholders, ", "), onConflict)

	batch := &pgx.Batch{}
	for _, row := range rows {
		batch.Queue(sql, row...)
	}
	br := tx.SendBatch(ctx, batch)
	defer func() { _ = br.Close() }()
	total := 0
	for range rows {
		ct, err := br.Exec()
		if err != nil {
			return total, err
		}
		total += int(ct.RowsAffected())
	}
	return total, br.Close()
}

// deleteScoped deletes child rows belonging to the given parent ids —
// parents not being promoted keep their children (additive-only guarantee).
func deleteScoped(ctx context.Context, tx pgx.Tx, table, parentCol string, parentIDs []string) error {
	if len(parentIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE %s = ANY($1)", ident(table), ident(parentCol)), parentIDs)
	return err
}

// rewriteURLs applies host replacements in-place to the named columns.
func rewriteURLs(rows [][]any, cols []column, urlCols []string, rewrites [][2]string) {
	for _, urlCol := range urlCols {
		i := colIndex(cols, urlCol)
		for _, row := range rows {
			s, ok := row[i].(string)
			if !ok { // NULL
				continue
			}
			for _, r := range rewrites {
				s = strings.ReplaceAll(s, r[0], r[1])
			}
			row[i] = s
		}
	}
}

// updateSetClause builds "col = EXCLUDED.col, ..." for every column not excluded.
func updateSetClause(cols []column, exclude ...string) string {
	var parts []string
	for _, c := range cols {
		if slices.Contains(exclude, c.name) {
			continue
		}
		parts = append(parts, ident(c.name)+" = EXCLUDED."+ident(c.name))
	}
	return strings.Join(parts, ", ")
}

func idsOf(rows [][]any, cols []column) []string {
	i := colIndex(cols, "id")
	ids := make([]string, len(rows))
	for j, row := range rows {
		id, ok := row[i].(string)
		if !ok {
			log.Fatalf("row %d has non-text id: %v", j, row[i])
		}
		ids[j] = id
	}
	return ids
}

func colIndex(cols []column, name string) int {
	for i, c := range cols {
		if c.name == name {
			return i
		}
	}
	log.Fatalf("column %q not found in %v", name, cols)
	return -1
}

func ident(name string) string {
	return pgx.Identifier{name}.Sanitize()
}

// dsnOrEnvFile returns $envVar, falling back to POSTGRES_DSN= in the given
// env file (same files make cdk-deploy-{env} sources).
func dsnOrEnvFile(envVar, file string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	data, err := os.ReadFile(file)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(line), "POSTGRES_DSN="); ok {
				return strings.Trim(v, `"'`)
			}
		}
	}
	log.Fatalf("%s not set and no POSTGRES_DSN in %s", envVar, file)
	return ""
}

// assetHost returns $envVar, falling back to the CloudFormation export that
// the storage stack publishes (the assets CloudFront distribution domain).
func assetHost(ctx context.Context, envVar, exportName, region string) string {
	host := os.Getenv(envVar)
	if host == "" {
		out, err := exec.CommandContext(ctx, "aws", "cloudformation", "list-exports", "--region", region, //nolint:gosec // fixed args; exportName is a compile-time constant, region validated in main
			"--query", fmt.Sprintf("Exports[?Name=='%s'].Value", exportName), "--output", "text").Output()
		if err != nil {
			log.Fatalf("resolve %s via CloudFormation export %s: %v (set %s to override)", envVar, exportName, err, envVar)
		}
		host = strings.TrimSpace(string(out))
	}
	if !regexp.MustCompile(`^[A-Za-z0-9.-]+$`).MatchString(host) {
		log.Fatalf("could not resolve asset host %s (got %q); set %s", //nolint:gosec // host is stripped to printable runes before logging
			exportName, strings.Map(printableOnly, host), envVar)
	}
	return host
}

// printableOnly strips control characters before a value is logged.
func printableOnly(r rune) rune {
	if r < 32 || r == 127 {
		return -1
	}
	return r
}

func confirm(prompt string) bool {
	fmt.Print(prompt)
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(answer) == "y"
}
