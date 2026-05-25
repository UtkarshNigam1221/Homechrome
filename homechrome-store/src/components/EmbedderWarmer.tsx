'use client';

import { useEffect } from 'react';

import { pingEmbedder } from '@/lib/semantic-search';

// EmbedderWarmer fires one /ping at the embedder Lambda when the storefront
// mounts in the browser. By the time the user composes a search query the
// container is warm.
//
// Single ping per page load — no interval. Users idling > ~15 min then
// searching accept a one-time 5–7 s cold start. The semanticSearch helper
// already retries once on 5xx as a safety net.
//
// Returns null (no UI). Mount once near the top of the App layout.
export default function EmbedderWarmer() {
  useEffect(() => {
    pingEmbedder();
  }, []);
  return null;
}
