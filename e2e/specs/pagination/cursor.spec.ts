import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient, json } from '../../fixtures/api';
import { createProduct, destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';

/**
 * #230 case 25 and #223 Tier 1.4.
 *
 * `QueryPage`/`QueryAll` were the fix for seven silently truncating queries and
 * have no direct tests — `grep -rn "QueryPage" --include='*_test.go'` returns
 * nothing. The filtered case is the one that matters: DynamoDB applies
 * FilterExpression *after* Limit, so a page can come back short, or empty, with
 * more rows still to come.
 */
test.describe('cursor pagination returns everything', () => {
  let api: APIRequestContext;
  let catalog: SeededCatalog | undefined;

  test.beforeAll(async () => {
    api = await adminClient();
  });

  test.afterAll(async () => {
    await destroyCatalog(api, catalog);
  });

  test('walking one row per page yields the same set as one large page', async () => {
    // Enough rows that a one-per-page walk is a real walk.
    catalog = await seedCatalog(api, [1, 1, 1]);
    for (let i = 0; i < 4; i++) {
      catalog.products.push(
        await createProduct(api, catalog.categoryId, { stock: 1, label: `page-${i}` })
      );
    }

    const single = await page(api, `/admin/products?category_id=${catalog.categoryId}&limit=100`);
    expect(single.ids.length, 'the seeded products should all be on one big page').toBe(
      catalog.products.length
    );

    const walked: string[] = [];
    let cursor: string | undefined;
    for (let guard = 0; guard < 50; guard++) {
      const qs = new URLSearchParams({ category_id: catalog.categoryId, limit: '1' });
      if (cursor) qs.set('cursor', cursor);
      const p = await page(api, `/admin/products?${qs.toString()}`);
      walked.push(...p.ids);
      if (!p.hasMore || !p.nextCursor) break;
      cursor = p.nextCursor;
    }

    expect(
      new Set(walked).size,
      'a one-per-page walk must not repeat or drop a row'
    ).toBe(walked.length);
    expect(walked.slice().sort()).toEqual(single.ids.slice().sort());
  });

  test('a filtered walk returns everything the single page does', async () => {
    // The dangerous case: the filter is applied after the limit, so an early
    // page can be empty while later ones still hold matches.
    const status = 'ACTIVE';
    const single = await page(api, `/admin/products?status=${status}&limit=100`);

    const walked: string[] = [];
    let cursor: string | undefined;
    for (let guard = 0; guard < 200; guard++) {
      const qs = new URLSearchParams({ status, limit: '2' });
      if (cursor) qs.set('cursor', cursor);
      const p = await page(api, `/admin/products?${qs.toString()}`);
      walked.push(...p.ids);
      if (!p.hasMore || !p.nextCursor) break;
      cursor = p.nextCursor;
    }

    for (const id of single.ids) {
      expect(walked, `filtered walk dropped ${id}`).toContain(id);
    }
    expect(new Set(walked).size, 'filtered walk must not repeat a row').toBe(walked.length);
  });
});

async function page(
  api: APIRequestContext,
  url: string
): Promise<{ ids: string[]; hasMore: boolean; nextCursor?: string }> {
  const body = await json<{
    products?: { id: string }[];
    pagination?: { has_more?: boolean; next_cursor?: string };
  }>(await api.get(url));

  return {
    ids: (body.products ?? []).map((p) => p.id),
    hasMore: !!body.pagination?.has_more,
    nextCursor: body.pagination?.next_cursor,
  };
}
