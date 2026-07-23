// Command promote-catalog copies catalog data (categories, products, media)
// from dev to prod. Additive-only: upserts by id, never deletes prod-only
// rows; child rows are replaced per promoted parent; asset URLs are rewritten
// to the prod CDN; S3 media syncs without --delete. Inventory is seeded only
// where missing unless --overwrite-inventory. DynamoDB data and
// inventory_transactions are never touched.
//
// Usage: go run ./scripts/promote-catalog [--products active|all|id1,id2,...]
// [--overwrite-inventory] [--skip-s3] [--yes]
//
// Env: DEV_DSN/PROD_DSN (default: POSTGRES_DSN from .env.{dev,prod}),
// DEV_ASSET_HOST/PROD_ASSET_HOST (default: CloudFormation exports), AWS_REGION.
package main

import (
	"bufio"
	"cmp"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	devBucket  = "handloom-assets-dev"
	prodBucket = "handloom-assets-prod"
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
	overwriteInventory := flag.Bool("overwrite-inventory", false, "overwrite prod stock with dev quantities (default: only seed missing rows)")
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

	productWhere, productArgs := productFilter(*products)
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

	// Schema parity: dev and prod must have identical columns, otherwise prod
	// needs a deploy (which runs the migrator) before promoting.
	tables := []string{
		"categories", "category_attributes", "category_attribute_options",
		"products", "product_attribute_values", "product_images", "inventory",
	}
	cols := map[string][]column{}
	for _, t := range tables {
		devCols, colErr := columnsOf(ctx, dev, t)
		if colErr != nil {
			log.Fatalf("read dev schema for %s: %v", t, colErr)
		}
		prodCols, colErr := columnsOf(ctx, prod, t)
		if colErr != nil {
			log.Fatalf("read prod schema for %s: %v", t, colErr)
		}
		if fmt.Sprint(devCols) != fmt.Sprint(prodCols) {
			log.Fatalf("schema drift on %q:\n  dev:  %v\n  prod: %v\ndeploy the backend to prod first (runs the migrator), then retry", t, devCols, prodCols)
		}
		cols[t] = devCols
	}

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

	inventoryMode := "seed missing rows only, prod stock untouched"
	if *overwriteInventory {
		inventoryMode = "OVERWRITE prod stock with dev values"
	}
	mediaMode := fmt.Sprintf("s3://%s/assets/ -> s3://%s/assets/ (additive)", devBucket, prodBucket)
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
	if !*skipS3 {
		log.Println("Syncing media...")
		sync := exec.CommandContext(ctx, "aws", "s3", "sync", //nolint:gosec // fixed args; region validated above
			"s3://"+devBucket+"/assets/", "s3://"+prodBucket+"/assets/", "--region", region)
		sync.Stdout, sync.Stderr = os.Stdout, os.Stderr
		if syncErr := sync.Run(); syncErr != nil {
			log.Fatalf("s3 sync: %v", syncErr)
		}
	}

	rewriteURLs(categories, cols["categories"], []string{"image_url"}, urlRewrites)
	rewriteURLs(prods, cols["products"], []string{"video_url", "video_poster_url"}, urlRewrites)
	rewriteURLs(images, cols["product_images"], []string{"url"}, urlRewrites)

	inventoryConflict := "ON CONFLICT (product_id) DO NOTHING"
	if *overwriteInventory {
		inventoryConflict = "ON CONFLICT (product_id) DO UPDATE SET " + updateSetClause(cols["inventory"], "id", "product_id")
	}

	// Single transaction: either the whole catalog lands or none of it.
	tx, err := prod.Begin(ctx)
	if err != nil {
		log.Fatalf("begin prod tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit

	step := func(name string, n int, err error) {
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		log.Printf("  %-28s %d rows", name, n)
	}

	// Categories: upsert by id. Prod-only categories untouched.
	n, err := insertRows(ctx, tx, "categories", cols["categories"], categories,
		"ON CONFLICT (id) DO UPDATE SET "+updateSetClause(cols["categories"], "id"))
	step("categories", n, err)

	// Category attributes + options: replace as a set, scoped to promoted
	// categories (options fall via ON DELETE CASCADE, then both re-insert).
	err = deleteScoped(ctx, tx, "category_attributes", "category_id", idsOf(categories, cols["categories"]))
	if err == nil {
		n, err = insertRows(ctx, tx, "category_attributes", cols["category_attributes"], categoryAttrs, "")
	}
	step("category_attributes", n, err)
	n, err = insertRows(ctx, tx, "category_attribute_options", cols["category_attribute_options"], categoryOpts, "")
	step("category_attribute_options", n, err)

	// Products: upsert by id. search_vector regenerates; embeddings copy over.
	n, err = insertRows(ctx, tx, "products", cols["products"], prods,
		"ON CONFLICT (id) DO UPDATE SET "+updateSetClause(cols["products"], "id"))
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
	fmt.Println("Note: prod catalog Lambda has a 1h in-process cache — force a cold start (or wait up to 1h) before expecting fresh data on the storefront.")
}

// productFilter maps the --products flag to a WHERE clause + args.
func productFilter(products string) (string, []any) {
	switch products {
	case "active":
		return "status = 'ACTIVE'", nil
	case "all":
		return "TRUE", nil
	default:
		ids := strings.Split(products, ",")
		for _, id := range ids {
			if !idListRe.MatchString(id) {
				log.Fatalf("--products must be \"active\", \"all\", or a comma-separated id list (bad id: %q)", id)
			}
		}
		return "id = ANY($1)", []any{ids}
	}
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
