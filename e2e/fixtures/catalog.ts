import { APIRequestContext } from '@playwright/test';

import { json } from './api';
import { RUN_ID, tag } from './run-id';

/**
 * The suite never touches real catalog products. It creates its own and deletes
 * them afterwards.
 *
 * This is not tidiness, it is the only exact restoration available.
 * `AdjustStock` sets `quantity` without ever writing `reserved_qty`
 * (inventory_repository.go), so compensating arithmetic through the stock API
 * carries any leak forward and would force falsifying the ledger. Deleting the
 * product is a real row delete that cascades `inventory` and
 * `inventory_transactions`, so a leaked reservation goes with it.
 */

export interface SeededProduct {
  id: string;
  sku: string;
  name: string;
  sellingPrice: number;
  initialStock: number;
}

export interface SeededCatalog {
  categoryId: string;
  products: SeededProduct[];
}

export async function createCategory(api: APIRequestContext): Promise<string> {
  const category = await json<{ id: string }>(
    await api.post('/admin/categories', {
      data: {
        name: tag('category'),
        description: `Created by the e2e suite, run ${RUN_ID}. Safe to delete.`,
      },
    })
  );
  return category.id;
}

export async function createProduct(
  api: APIRequestContext,
  categoryId: string,
  opts: { stock: number; price?: number; label?: string }
): Promise<SeededProduct> {
  const name = tag(opts.label ?? 'product');
  const sellingPrice = opts.price ?? 30_000; // ₹300 in paise

  const product = await json<{ id: string; sku: string }>(
    await api.post('/admin/products', {
      data: {
        name,
        sku: name,
        category_id: categoryId,
        description: 'e2e fixture',
        base_price: sellingPrice,
        selling_price: sellingPrice,
        initial_stock: opts.stock,
        low_stock_threshold: 0,
        status: 'ACTIVE',
      },
    })
  );

  return {
    id: product.id,
    sku: product.sku ?? name,
    name,
    sellingPrice,
    initialStock: opts.stock,
  };
}

/** One category plus n products, each independently stocked. */
export async function seedCatalog(
  api: APIRequestContext,
  stocks: number[]
): Promise<SeededCatalog> {
  const categoryId = await createCategory(api);
  const products: SeededProduct[] = [];
  for (const [i, stock] of stocks.entries()) {
    products.push(await createProduct(api, categoryId, { stock, label: `product-${i + 1}` }));
  }
  return { categoryId, products };
}

/**
 * Order matters: a category refuses deletion while it still has products, so
 * products go first. Failures are swallowed — teardown must not turn a passing
 * run red, and scripts/cleanup.ts sweeps whatever is left.
 */
export async function destroyCatalog(
  api: APIRequestContext,
  catalog: SeededCatalog | undefined
): Promise<void> {
  if (!catalog) return;

  for (const product of catalog.products) {
    try {
      await api.delete(`/admin/products/${product.id}`);
    } catch {
      // swept by scripts/cleanup.ts
    }
  }
  try {
    await api.delete(`/admin/categories/${catalog.categoryId}`);
  } catch {
    // swept by scripts/cleanup.ts
  }
}
