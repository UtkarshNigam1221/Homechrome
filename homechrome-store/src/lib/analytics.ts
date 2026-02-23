const BATCH_SIZE = 10;
const FLUSH_INTERVAL_MS = 30_000;
const API_URL = '/api/v1/store/events';

interface TrackingEvent {
  event_type: string;
  timestamp: string;
  session_id: string;
  visitor_id: string;
  device_type: 'mobile' | 'desktop' | 'tablet';
  page_path: string;
  properties: Record<string, unknown>;
}

const eventBuffer: TrackingEvent[] = [];
let flushTimer: ReturnType<typeof setInterval> | null = null;
let initialized = false;

function getSessionId(): string {
  let id = sessionStorage.getItem('hc_session_id');
  if (!id) {
    id = crypto.randomUUID();
    sessionStorage.setItem('hc_session_id', id);
  }
  return id;
}

function getVisitorId(): string {
  let id = localStorage.getItem('hc_visitor_id');
  if (!id) {
    id = crypto.randomUUID();
    localStorage.setItem('hc_visitor_id', id);
  }
  return id;
}

function getDeviceType(): 'mobile' | 'desktop' | 'tablet' {
  const width = window.innerWidth;
  if (width < 768) return 'mobile';
  if (width < 1024) return 'tablet';
  return 'desktop';
}

function flush() {
  if (eventBuffer.length === 0) return;

  const batch = eventBuffer.splice(0);
  const body = JSON.stringify({ events: batch });

  // Use sendBeacon for reliability (works during page unload)
  if (navigator.sendBeacon) {
    navigator.sendBeacon(
      API_URL,
      new Blob([body], { type: 'application/json' }),
    );
  } else {
    fetch(API_URL, {
      method: 'POST',
      body,
      headers: { 'Content-Type': 'application/json' },
      keepalive: true,
    });
  }
}

export function track(
  eventType: string,
  properties: Record<string, unknown> = {},
) {
  if (typeof window === 'undefined') return;

  const event: TrackingEvent = {
    event_type: eventType,
    timestamp: new Date().toISOString(),
    session_id: getSessionId(),
    visitor_id: getVisitorId(),
    device_type: getDeviceType(),
    page_path: window.location.pathname,
    properties,
  };

  eventBuffer.push(event);

  if (eventBuffer.length >= BATCH_SIZE) {
    flush();
  }
}

export function initAnalytics() {
  if (typeof window === 'undefined') return;
  if (initialized) return;
  initialized = true;

  // Periodic flush
  flushTimer = setInterval(flush, FLUSH_INTERVAL_MS);

  // Flush on page unload
  window.addEventListener('beforeunload', flush);

  // Flush on visibility change (tab backgrounded)
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') flush();
  });
}

export function stopAnalytics() {
  if (flushTimer) clearInterval(flushTimer);
  flushTimer = null;
  initialized = false;
  flush();
}
