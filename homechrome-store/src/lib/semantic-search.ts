// Semantic-search client that talks directly to the embedder Lambda's
// Function URL. When the flag is off, the higher-level `searchProducts`
// helper in `api.ts` falls back to the legacy keyword endpoint.

const SEMANTIC = process.env.NEXT_PUBLIC_SEMANTIC_SEARCH_ENABLED === 'true';
const URL_BASE = process.env.NEXT_PUBLIC_SEMANTIC_SEARCH_URL ?? '';

export type SearchProduct = {
  id: string;
  name: string;
  slug: string;
  price_paise: number;
  primary_image_url: string;
};

export type SearchResponse = {
  success: boolean;
  data: SearchProduct[];
  meta: {
    limit: number;
    offset: number;
    total_estimate: number;
  };
};

export function isSemanticEnabled(): boolean {
  return SEMANTIC && URL_BASE !== '';
}

async function postSearch(q: string, limit: number, offset: number): Promise<Response> {
  return fetch(`${URL_BASE}/search`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ q, limit, offset }),
  });
}

/**
 * Hits the embedder Function URL. Retries once on 5xx with a 1s back-off to
 * cover cold-start blips. Throws on network error after the retry.
 */
export async function semanticSearch(
  q: string,
  limit = 20,
  offset = 0,
): Promise<SearchResponse> {
  let res = await postSearch(q, limit, offset);
  if (!res.ok && res.status >= 500) {
    await new Promise((r) => setTimeout(r, 1000));
    res = await postSearch(q, limit, offset);
  }
  if (!res.ok) {
    throw new Error(`semantic search failed: ${res.status}`);
  }
  return (await res.json()) as SearchResponse;
}

/**
 * Best-effort warmup. Called on page mount via EmbedderWarmer (T25). Failure
 * is swallowed — the user simply pays a cold start on their first /search.
 */
export async function pingEmbedder(): Promise<void> {
  if (!isSemanticEnabled()) return;
  try {
    await fetch(`${URL_BASE}/ping`, { method: 'GET', keepalive: true });
  } catch {
    // best-effort
  }
}
