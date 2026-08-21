const BASE_DOMAIN = 'homechrome.in';
export const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL || `https://${BASE_DOMAIN}`;
// Only the production canonical host (apex or www) is crawlable/indexable. Dev
// (dev-store.*), preview, and localhost serve Disallow + noindex so they never
// compete with prod in search.
export const IS_INDEXABLE = /^https:\/\/(www\.)?homechrome\.in(\/|$)/.test(SITE_URL);
export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';

// Catalog page size (SSR first page + scroll pages); backend caps limit at 100.
export const PRODUCTS_PAGE_SIZE = 24;

// Support contacts. One number drives all three: display, tel: (wants the
// +country form) and wa.me (wants bare digits). Env-overridable per environment.
export const SUPPORT_PHONE = process.env.NEXT_PUBLIC_SUPPORT_PHONE || '+91-6396223764';
export const SUPPORT_WHATSAPP = SUPPORT_PHONE.replace(/\D/g, '');
export const SUPPORT_PHONE_TEL = `+${SUPPORT_WHATSAPP}`;
export const SUPPORT_EMAIL = 'info@homechrome.in';
export const INSTAGRAM_URL = 'https://www.instagram.com/_home.chrome_';

// Legal entity + policy facts — single source of truth for the legal pages
// ((legal) route group, contact grievance section). Update LEGAL_LAST_UPDATED
// whenever any policy text or these values change.
export const LEGAL_ENTITY_NAME = 'Home Shome';
export const LEGAL_PROPRIETOR = 'Ritu Batra';
export const LEGAL_ADDRESS =
  '945, Shri Nagar, Railway Road, Hapur, Uttar Pradesh 245101, India';
export const GSTIN = '[GSTIN to be updated]';
export const GRIEVANCE_OFFICER = 'Ritu Batra';
export const DAMAGE_CLAIM_WINDOW_HOURS = 48;
export const DISPATCH_DAYS = '2–3';
export const DELIVERY_DAYS = '5–10';
export const REFUND_DAYS = '5–7';
export const LEGAL_LAST_UPDATED = '21 July 2026';
