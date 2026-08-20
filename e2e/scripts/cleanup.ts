/**
 * Idempotent reaper for anything this suite created, in any run.
 *
 * Deliberately separate from Playwright teardown: a crashed spec skips its own
 * afterEach, and a cancelled CI job skips everything. Runs as a post-step with
 * `if: always()`, and can be run by hand.
 *
 *   npm run cleanup            # delete E2E- products and categories
 *   DRY_RUN=1 npm run cleanup  # list what would go, delete nothing
 *
 * Products go before categories: a category refuses deletion while it still
 * holds products. Deleting a product is a real row delete that cascades
 * inventory and inventory_transactions, so a leaked reservation goes with it —
 * which is the only exact restoration available, since AdjustStock cannot write
 * reserved_qty.
 *
 * Orders are not deleted. OrderRepository has no Delete method at all; cancel
 * is the only lever and the specs already use it.
 */
import { request } from '@playwright/test';

const E2E_PREFIX = 'E2E-';
const DRY_RUN = !!process.env.DRY_RUN;

interface Named {
  id: string;
  name: string;
}

function required(name: string): string {
  const v = process.env[name];
  if (!v) throw new Error(`${name} must be set`);
  return v;
}

async function main(): Promise<void> {
  const api = await request.newContext({ baseURL: required('API_URL') });

  const login = await api.post('/admin/auth/login', {
    data: {
      email: required('E2E_ADMIN_EMAIL'),
      password: required('E2E_ADMIN_PASSWORD'),
    },
  });
  if (!login.ok()) {
    throw new Error(`admin login failed: ${login.status()} ${await login.text()}`);
  }

  const products = await listAll<Named>(api, '/admin/products', 'products');
  const staleProducts = products.filter((p) => p.name?.startsWith(E2E_PREFIX));

  const categories = await listAll<Named>(api, '/admin/categories', 'categories');
  const staleCategories = categories.filter((c) => c.name?.startsWith(E2E_PREFIX));

  console.log(
    `found ${staleProducts.length} e2e product(s) and ${staleCategories.length} e2e category(ies)`
  );

  let failures = 0;

  for (const product of staleProducts) {
    if (DRY_RUN) {
      console.log(`would delete product ${product.id} ${product.name}`);
      continue;
    }
    const res = await api.delete(`/admin/products/${product.id}`);
    if (res.ok()) {
      console.log(`deleted product ${product.name}`);
    } else {
      failures++;
      console.error(`could not delete product ${product.name}: ${res.status()} ${await res.text()}`);
    }
  }

  for (const category of staleCategories) {
    if (DRY_RUN) {
      console.log(`would delete category ${category.id} ${category.name}`);
      continue;
    }
    const res = await api.delete(`/admin/categories/${category.id}`);
    if (res.ok()) {
      console.log(`deleted category ${category.name}`);
    } else {
      // Usually "still has products" — a product delete above failed, and the
      // next run will get both. Reported, not fatal.
      failures++;
      console.error(
        `could not delete category ${category.name}: ${res.status()} ${await res.text()}`
      );
    }
  }

  await api.dispose();

  if (failures > 0) {
    console.error(`${failures} entity(ies) could not be removed; re-run cleanup`);
    process.exitCode = 1;
  }
}

/** Walks the cursor until it runs out. A single large page would truncate. */
async function listAll<T>(
  api: Awaited<ReturnType<typeof request.newContext>>,
  path: string,
  key: string
): Promise<T[]> {
  const out: T[] = [];
  let cursor: string | undefined;

  for (let page = 0; page < 50; page++) {
    const qs = new URLSearchParams({ limit: '100' });
    if (cursor) qs.set('cursor', cursor);

    const res = await api.get(`${path}?${qs.toString()}`);
    if (!res.ok()) break;

    // The two list endpoints do not agree on a shape, so handle both rather
    // than silently finding nothing:
    //
    //   /admin/products   → response.JSON(result)          → {products, pagination}
    //   /admin/categories → response.SuccessWithMeta(list) → {data: [...], meta}
    //
    // Assuming the first is why cleanup reported "0 e2e category(ies)" while an
    // orphaned category sat in dev holding its slug against the next run.
    const body = (await res.json()) as Record<string, unknown> & {
      data?: unknown;
      meta?: { next_cursor?: string; has_more?: boolean };
    };
    const unwrapped = body.data ?? body;

    let items: T[];
    let pagination: { next_cursor?: string; has_more?: boolean } | undefined;

    if (Array.isArray(unwrapped)) {
      items = unwrapped as T[];
      pagination = body.meta;
    } else {
      const payload = unwrapped as Record<string, unknown>;
      items = (payload[key] ?? []) as T[];
      pagination = payload.pagination as typeof pagination;
    }

    out.push(...items);

    if (!pagination?.has_more || !pagination.next_cursor) break;
    cursor = pagination.next_cursor;
  }
  return out;
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
