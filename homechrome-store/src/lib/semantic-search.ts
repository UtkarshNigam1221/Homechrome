// Warmer for the embedder Lambda. The search call itself lives in api.ts
// (fetchProductsPage) and goes through the existing REST API.

import { ROUTES } from '@/lib/routes';

/**
 * Best-effort warmup of the embedder Lambda. Called on page mount by
 * EmbedderWarmer so the user's first /search doesn't pay a cold start.
 * Failures are swallowed — worst case the user waits ~5-7 s once.
 *
 * No credentials — the embedder doesn't read cookies and its CORS config
 * sets AllowCredentials:false, so a credentialed request would be
 * rejected outright in any cross-origin scenario (e.g. local dev without
 * the Next.js rewrite proxy).
 */
export async function pingEmbedder(): Promise<void> {
  try {
    await fetch(ROUTES.CATALOG.EMBEDDER_PING, {
      method: 'GET',
      keepalive: true,
    });
  } catch {
    // best-effort
  }
}
