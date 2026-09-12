import { ROUTES } from "@/lib/routes";
import {
  buildVisitorHeader,
  captureUTMFromURL,
  getDeviceType as resolveDeviceType,
  VISITOR_HEADER,
} from "@/lib/visitor-context";

const BATCH_SIZE = 10;
const FLUSH_INTERVAL_MS = 30_000;
const API_URL = ROUTES.EVENTS;

// Per-request body budget. The Fetch spec caps total in-flight keepalive body
// size at 64 KB per origin; a large unload/visibility flush above that is
// silently dropped. We chunk each flush so no single request exceeds this.
const MAX_KEEPALIVE_BYTES = 60_000;

// Kill-switch: default on. Set NEXT_PUBLIC_ANALYTICS_ENABLED=false to disable.
const ENABLED = process.env.NEXT_PUBLIC_ANALYTICS_ENABLED !== "false";

interface TrackingEvent {
  event_type: string;
  timestamp: string;
  session_id: string;
  visitor_id: string;
  device_type: "mobile" | "desktop" | "tablet";
  page_path: string;
  properties: Record<string, unknown>;
}

const eventBuffer: TrackingEvent[] = [];
let flushTimer: ReturnType<typeof setInterval> | null = null;
let initialized = false;

// crypto.randomUUID is exposed only in secure contexts, so it is absent when
// the site is served over plain HTTP — a phone hitting the dev server by LAN
// IP. getRandomValues is available there, so build the v4 by hand instead.
function uuid(): string {
  if (typeof crypto.randomUUID === "function") return crypto.randomUUID();

  const bytes = crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 0x0f) | 0x40; // version 4
  bytes[8] = (bytes[8] & 0x3f) | 0x80; // variant 1
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

function getSessionId(): string {
  let id = sessionStorage.getItem("hc_session_id");
  if (!id) {
    id = uuid();
    sessionStorage.setItem("hc_session_id", id);
  }
  return id;
}

function getVisitorId(): string {
  let id = localStorage.getItem("hc_visitor_id");
  if (!id) {
    id = uuid();
    localStorage.setItem("hc_visitor_id", id);
  }
  return id;
}

function flush() {
  if (eventBuffer.length === 0) return;

  const batch = eventBuffer.splice(0);

  // fetch+keepalive instead of navigator.sendBeacon because sendBeacon
  // cannot set custom request headers (spec-level limitation) and we
  // need X-Hc-Visitor on this beacon so the backend tags every
  // site_visitor / rum_page_view / product_viewed row with the visitor's
  // device + sticky UTM tuple. keepalive: true gives near-equivalent
  // page-unload reliability across modern browsers.
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  const visitor = buildVisitorHeader();
  if (visitor) headers[VISITOR_HEADER] = visitor;

  for (const chunk of chunkBySize(batch)) {
    fetch(API_URL, {
      method: "POST",
      body: JSON.stringify({ events: chunk }),
      headers,
      keepalive: true,
    }).catch(() => {
      // Beacon endpoint is fire-and-forget — never let a network error
      // surface to the user.
    });
  }
}

// chunkBySize splits events into groups whose serialized body stays under the
// keepalive limit. An event larger than the budget on its own is still sent
// alone rather than dropping the whole batch.
function chunkBySize(events: TrackingEvent[]): TrackingEvent[][] {
  const chunks: TrackingEvent[][] = [];
  let current: TrackingEvent[] = [];
  let size = 0;
  for (const ev of events) {
    const evSize = JSON.stringify(ev).length + 1; // + separator
    if (current.length > 0 && size + evSize > MAX_KEEPALIVE_BYTES) {
      chunks.push(current);
      current = [];
      size = 0;
    }
    current.push(ev);
    size += evSize;
  }
  if (current.length > 0) chunks.push(current);
  return chunks;
}

export function track(
  eventType: string,
  properties: Record<string, unknown> = {},
) {
  if (!ENABLED) return;
  if (typeof window === "undefined") return;

  const event: TrackingEvent = {
    event_type: eventType,
    timestamp: new Date().toISOString(),
    session_id: getSessionId(),
    visitor_id: getVisitorId(),
    device_type: resolveDeviceType(),
    page_path: window.location.pathname,
    properties,
  };

  eventBuffer.push(event);

  if (eventBuffer.length >= BATCH_SIZE) {
    flush();
  }
}

function handleVisibilityChange() {
  if (document.visibilityState === "hidden") flush();
}

export function initAnalytics() {
  if (!ENABLED) return;
  if (typeof window === "undefined") return;
  if (initialized) return;
  initialized = true;

  // Sticky-attribution UTM capture. Runs once per page load — if any
  // ?utm_* params are present they overwrite the last stored set so
  // latest-touch attribution wins (matches GA / Mixpanel default).
  captureUTMFromURL();

  flushTimer = setInterval(flush, FLUSH_INTERVAL_MS);
  window.addEventListener("beforeunload", flush);
  document.addEventListener("visibilitychange", handleVisibilityChange);
}

export function stopAnalytics() {
  if (flushTimer) clearInterval(flushTimer);
  flushTimer = null;
  initialized = false;
  window.removeEventListener("beforeunload", flush);
  document.removeEventListener("visibilitychange", handleVisibilityChange);
  flush();
}
