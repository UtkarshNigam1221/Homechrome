const BASE_DOMAIN = 'homechrome.in';
export const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL || `https://${BASE_DOMAIN}`;
// Only the production canonical host (apex or www) is crawlable/indexable. Dev
// (dev-store.*), preview, and localhost serve Disallow + noindex so they never
// compete with prod in search.
export const IS_INDEXABLE = /^https:\/\/(www\.)?homechrome\.in(\/|$)/.test(SITE_URL);
export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';

// Support / social contacts. wa.me wants digits only (incl. country code);
// phone stays env-overridable (placeholder number for now).
export const SUPPORT_WHATSAPP = (
  process.env.NEXT_PUBLIC_SUPPORT_PHONE || '+919690965200'
).replace(/\D/g, '');
export const SUPPORT_EMAIL = 'info@homechrome.in';
export const INSTAGRAM_URL = 'https://www.instagram.com/_home.chrome_';
