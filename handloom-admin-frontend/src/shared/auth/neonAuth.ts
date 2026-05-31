import { createInternalNeonAuth } from '@neondatabase/neon-js/auth';

// Neon Auth (Better Auth) — separate login alongside admin auth.
// Used ONLY to mint short-lived JWTs for Neon Data API requests.
// Existing admin custom-JWT auth (LoginPage + Zustand) is unaffected.

const baseURL = import.meta.env.VITE_NEON_AUTH_URL ?? '';

if (!baseURL) {
  console.warn('VITE_NEON_AUTH_URL not set — Neon Auth + Data API dashboards will not work.');
}

// createInternalNeonAuth returns { adapter, getJWTToken }; createAuthClient
// only returns the adapter (no getJWTToken). We need the token for Data API.
const neon = createInternalNeonAuth(baseURL);

// Public Better Auth client (signIn.social, signOut, getSession, useSession, …).
export const neonAuthClient = neon.adapter;

// JWT TTL is 15 min in Neon Auth — cache for 12 min, then re-mint.
let cachedToken: { value: string; expiresAt: number } | null = null;

/**
 * Returns the current Neon Auth JWT, or null if no session.
 * Caller (Data API axios interceptor) attaches it as `Authorization: Bearer …`.
 */
export async function getNeonAuthToken(): Promise<string | null> {
  if (cachedToken && Date.now() < cachedToken.expiresAt) {
    return cachedToken.value;
  }
  try {
    const token = await neon.getJWTToken();
    if (!token) {
      cachedToken = null;
      return null;
    }
    cachedToken = { value: token, expiresAt: Date.now() + 12 * 60 * 1000 };
    return token;
  } catch (err) {
    console.warn('Neon Auth token fetch failed:', err);
    cachedToken = null;
    return null;
  }
}

export function clearNeonAuthCache(): void {
  cachedToken = null;
}
