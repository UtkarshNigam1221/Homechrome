'use client';

import { onCLS, onINP, onLCP, onTTFB, Metric } from 'web-vitals';

import { track } from '@/lib/analytics';

function pageType(path: string): string {
  if (path === '/') return 'home';
  if (path.startsWith('/c/')) return 'category';
  if (path.startsWith('/p/')) return 'pdp';
  if (path.startsWith('/cart')) return 'cart';
  if (path.startsWith('/checkout')) return 'checkout';
  if (path.startsWith('/account')) return 'account';
  if (path.startsWith('/login')) return 'login';
  if (path.startsWith('/track')) return 'track';
  return 'other';
}

function send(eventType: string, value: number) {
  track(eventType, {
    value,
    page_type: pageType(window.location.pathname),
  });
}

let initialized = false;

export function initRUM() {
  if (initialized) return;
  if (typeof window === 'undefined') return;
  initialized = true;

  onLCP((m) => send('rum_lcp', m.value));
  onINP((m) => send('rum_inp', m.value));
  onCLS((m: Metric) => send('rum_cls', m.value * 100));
  onTTFB((m) => send('rum_ttfb', m.value));

  window.addEventListener('error', (ev) => {
    track('rum_js_error', {
      page_type: pageType(window.location.pathname),
      error_type: (ev.error?.name || 'Error').substring(0, 64),
    });
  });

  window.addEventListener('unhandledrejection', (ev) => {
    const reason = ev.reason as { name?: string } | undefined;
    const name = reason?.name || 'Error';
    track('rum_js_error', {
      page_type: pageType(window.location.pathname),
      error_type: ('unhandled_rejection_' + name).substring(0, 64),
    });
  });

  track('rum_page_view', { page_type: pageType(window.location.pathname) });
}
