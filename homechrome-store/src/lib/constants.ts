const BASE_DOMAIN = 'homechrome.in';
export const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL || `https://${BASE_DOMAIN}`;
export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';

// Support contact — configurable via env; placeholder number for now.
export const SUPPORT_PHONE = process.env.NEXT_PUBLIC_SUPPORT_PHONE || '+919690965200';
export const SUPPORT_PHONE_DISPLAY =
  process.env.NEXT_PUBLIC_SUPPORT_PHONE_DISPLAY || '+91 96909 65200';
// wa.me wants digits only, including country code.
export const SUPPORT_WHATSAPP = SUPPORT_PHONE.replace(/\D/g, '');
export const SUPPORT_EMAIL = process.env.NEXT_PUBLIC_SUPPORT_EMAIL || 'info@homechrome.in';

export const INSTAGRAM_URL =
  process.env.NEXT_PUBLIC_INSTAGRAM_URL || 'https://www.instagram.com/_home.chrome_';
