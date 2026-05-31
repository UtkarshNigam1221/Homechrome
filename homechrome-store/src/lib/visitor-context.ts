// visitor-context.ts
//
// Browser-side helpers for visitor-attribution headers that the backend
// labels metrics with (X-Device-Type + X-Utm-*). UTM params are sticky:
// captured from the URL on first visit, persisted in localStorage, and
// resent on every API request so funnel attribution survives cross-page
// navigation, OTP flows, etc.

const UTM_STORAGE_KEY = 'hc_utm';
const UTM_MAX_LEN = 32;

export interface UTMValues {
  utm_source: string;
  utm_medium: string;
  utm_campaign: string;
}

export type DeviceType = 'mobile' | 'tablet' | 'desktop';

function truncate(s: string): string {
  return s.toLowerCase().trim().slice(0, UTM_MAX_LEN);
}

/**
 * If the current URL has any utm_* params, write them to localStorage so
 * later requests can include them as headers. Idempotent — safe to call
 * on every page load. New utm values overwrite older ones (latest-touch
 * wins) which matches the typical marketing-attribution expectation.
 */
export function captureUTMFromURL(): void {
  if (typeof window === 'undefined') return;
  const params = new URLSearchParams(window.location.search);
  const source = params.get('utm_source');
  const medium = params.get('utm_medium');
  const campaign = params.get('utm_campaign');
  if (!source && !medium && !campaign) return;

  const stored: UTMValues = {
    utm_source: source ? truncate(source) : '',
    utm_medium: medium ? truncate(medium) : '',
    utm_campaign: campaign ? truncate(campaign) : '',
  };
  try {
    localStorage.setItem(UTM_STORAGE_KEY, JSON.stringify(stored));
  } catch {
    // localStorage may be disabled / over-quota — ignore.
  }
}

/**
 * Reads the sticky UTM tuple from localStorage. Returns empty strings
 * when nothing was ever captured; the backend treats those as "unknown".
 */
export function getUTM(): UTMValues {
  if (typeof window === 'undefined') return { utm_source: '', utm_medium: '', utm_campaign: '' };
  try {
    const raw = localStorage.getItem(UTM_STORAGE_KEY);
    if (!raw) return { utm_source: '', utm_medium: '', utm_campaign: '' };
    const parsed = JSON.parse(raw) as Partial<UTMValues>;
    return {
      utm_source: parsed.utm_source ?? '',
      utm_medium: parsed.utm_medium ?? '',
      utm_campaign: parsed.utm_campaign ?? '',
    };
  } catch {
    return { utm_source: '', utm_medium: '', utm_campaign: '' };
  }
}

/**
 * Coarse device-type classification driven by window width. Shared with
 * the analytics beacon so both backend-attached labels and the storefront
 * event stream agree on what device a session was on.
 */
export function getDeviceType(): DeviceType {
  if (typeof window === 'undefined') return 'desktop';
  const width = window.innerWidth;
  if (width < 768) return 'mobile';
  if (width < 1024) return 'tablet';
  return 'desktop';
}

/**
 * VISITOR_HEADER is the single request header that carries every visitor-
 * attribution field as `key=value;key=value;...`. Single-header design keeps
 * the CloudFront OriginRequestPolicy AllowList under the 10-header default
 * cap and gives us room to grow without re-touching CDK every time.
 */
export const VISITOR_HEADER = 'X-Hc-Visitor';

/**
 * Pack the browser-known fields (device + sticky UTM tuple) into the
 * X-Hc-Visitor header value. Empty values are skipped. Values are URL-
 * encoded so `;`, `=`, or `,` inside campaign names don't corrupt the
 * pair-splitter on the backend.
 */
export function buildVisitorHeader(): string {
  if (typeof window === 'undefined') return '';
  const parts: string[] = [];
  parts.push(`device=${encodeURIComponent(getDeviceType())}`);
  const utm = getUTM();
  if (utm.utm_source) parts.push(`utm_source=${encodeURIComponent(utm.utm_source)}`);
  if (utm.utm_medium) parts.push(`utm_medium=${encodeURIComponent(utm.utm_medium)}`);
  if (utm.utm_campaign) parts.push(`utm_campaign=${encodeURIComponent(utm.utm_campaign)}`);
  return parts.join(';');
}
