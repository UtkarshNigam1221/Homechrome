#!/usr/bin/env bash
#
# Clears transactional data before deploying the inventory-integrity branch,
# while leaving the catalog alone.
#
# Two key shapes change in this branch — payments move to MERCHANT_TXN#<id> and
# customer email lookup moves from a GSI to a pointer item — and SetKeys() only
# runs on write, so rows written before the deploy keep the old keys and become
# unreachable. Rather than backfill, dev drops the data that carries them.
#
# KEEPS:  categories, products, product images and attributes, inventory levels,
#         and manual stock movements (ADD / REMOVE / ADJUST).
# DROPS:  everything in the orders table — customers, orders, payments, refunds,
#         carts, quotes, shipments — and the order-scoped stock reservations
#         that referenced them.
#
# Usage:  AWS_PROFILE=... TABLE=handloom-orders-dev PGURL=postgres://... ./reset-transactional-data.sh
#         Add ENDPOINT=http://localhost:4566 for LocalStack.
set -euo pipefail

TABLE="${TABLE:?set TABLE, e.g. handloom-orders-dev}"
PGURL="${PGURL:?set PGURL to the catalog database}"
REGION="${REGION:-ap-south-1}"
ENDPOINT_ARG=""
[ -n "${ENDPOINT:-}" ] && ENDPOINT_ARG="--endpoint-url=$ENDPOINT"

say() { printf '\n\033[1m%s\033[0m\n' "$*"; }

say "Before"
psql "$PGURL" -tAc "SELECT '  categories: '||COUNT(*) FROM categories"
psql "$PGURL" -tAc "SELECT '  products:   '||COUNT(*) FROM products"
psql "$PGURL" -tAc "SELECT '  reserved:   '||COALESCE(SUM(reserved_qty),0)||' units across '||COUNT(*)||' rows' FROM inventory"
aws $ENDPOINT_ARG dynamodb scan --region "$REGION" --table-name "$TABLE" \
  --select COUNT --query 'Count' --output text | sed 's/^/  orders table items: /'

say "1/2  Emptying $TABLE"
# Loops until the table reports empty rather than deleting one scan's worth.
# A single pass would miss two things: a scan larger than one page, and the
# UnprocessedItems BatchWriteItem returns when it throttles. Both fail silently,
# and a payment left behind is one the new key shape can no longer find.
total=0
for attempt in $(seq 1 40); do
  remaining=$(aws $ENDPOINT_ARG dynamodb scan --region "$REGION" --table-name "$TABLE" \
    --select COUNT --query 'Count' --output text)
  [ "$remaining" = "0" ] && break

  deleted=$(aws $ENDPOINT_ARG dynamodb scan --region "$REGION" --table-name "$TABLE" \
      --projection-expression "PK,SK" --max-items 500 --output json \
    | python3 -c '
import json, subprocess, sys, os
items = json.load(sys.stdin)["Items"]
table, region = os.environ["TABLE"], os.environ.get("REGION", "ap-south-1")
endpoint = os.environ.get("ENDPOINT")
base = ["aws"] + ([f"--endpoint-url={endpoint}"] if endpoint else []) + ["dynamodb"]
done = 0
for i in range(0, len(items), 25):
    request = {table: [{"DeleteRequest": {"Key": it}} for it in items[i:i + 25]]}
    # Retry whatever BatchWriteItem hands back unprocessed, or those rows stay.
    for _ in range(6):
        out = subprocess.run(base + ["batch-write-item", "--region", region,
                                     "--request-items", json.dumps(request)],
                             check=True, capture_output=True, text=True)
        request = (json.loads(out.stdout or "{}").get("UnprocessedItems") or {})
        if not request:
            break
    else:
        sys.exit("BatchWriteItem kept returning unprocessed items")
    done += len(items[i:i + 25])
print(done)
')
  total=$((total + deleted))
  printf '  deleted %s (running total %s)\n' "$deleted" "$total"
done

final=$(aws $ENDPOINT_ARG dynamodb scan --region "$REGION" --table-name "$TABLE" \
  --select COUNT --query 'Count' --output text)
if [ "$final" != "0" ]; then
  echo "  ERROR: $final items still present after 40 passes" >&2
  exit 1
fi

say "2/2  Releasing the reservations those orders held"
# The orders are gone, so nothing can ever settle their reservations: the
# reconciliation report would flag them forever and the stock would stay held.
# Manual movements are kept — they reference no order and are the admin's own
# stocktake history.
psql "$PGURL" <<'SQL'
BEGIN;
DELETE FROM inventory_transactions WHERE reference_type = 'ORDER';
UPDATE inventory
   SET reserved_qty  = 0,
       available_qty = quantity,
       updated_at    = now()
 WHERE reserved_qty <> 0;
COMMIT;
SQL

say "After"
psql "$PGURL" -tAc "SELECT '  categories: '||COUNT(*) FROM categories"
psql "$PGURL" -tAc "SELECT '  products:   '||COUNT(*) FROM products"
psql "$PGURL" -tAc "SELECT '  reserved:   '||COALESCE(SUM(reserved_qty),0)||' units' FROM inventory"
psql "$PGURL" -tAc "SELECT '  manual ledger rows kept: '||COUNT(*) FROM inventory_transactions"
aws $ENDPOINT_ARG dynamodb scan --region "$REGION" --table-name "$TABLE" \
  --select COUNT --query 'Count' --output text | sed 's/^/  orders table items: /'

say "Done. Deploy now — the migrator applies 013 and 014 on start."
