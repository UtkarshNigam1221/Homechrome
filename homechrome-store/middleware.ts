// homechrome-store/middleware.ts
//
// Next.js middleware that reads CloudFront viewer geo headers (country,
// city, lat, lng) and merges them into the single X-Hc-Visitor request
// header before the /api/* rewrite forwards to backend.
//
// The browser-side axios interceptor (lib/api.ts) sets X-Hc-Visitor with
// the browser-known fields (device + sticky UTM tuple). The CloudFront
// viewer-geo bits are only visible server-side, so this middleware is
// the one place where they get folded into the same header. Backend
// then sees a single tuple-packed header carrying every attribution
// label needed to tag metrics.
//
// Header shape: `key=value;key=value;...`. Values are URL-encoded so
// campaign names with `;`, `=`, or `,` are safe.

import { NextRequest, NextResponse } from "next/server";

import { VISITOR_HEADER } from "@/lib/visitor-context";

export function middleware(req: NextRequest) {
  const country = (req.headers.get("cloudfront-viewer-country") ?? "")
    .toUpperCase()
    .trim();
  const city = (req.headers.get("cloudfront-viewer-city") ?? "")
    .toLowerCase()
    .trim();
  const lat = req.headers.get("cloudfront-viewer-latitude") ?? "";
  const lng = req.headers.get("cloudfront-viewer-longitude") ?? "";

  const incoming = req.headers.get(VISITOR_HEADER) ?? "";
  const parts: string[] = incoming
    ? incoming
        .split(";")
        .map((s) => s.trim())
        .filter(Boolean)
    : [];

  const setField = (key: string, value: string) => {
    if (!value) return;
    // Drop any browser-set value for the same key — server-resolved geo wins.
    const filtered = parts.filter((p) => !p.startsWith(`${key}=`));
    parts.length = 0;
    parts.push(...filtered);
    parts.push(`${key}=${encodeURIComponent(value)}`);
  };

  setField("city", city);
  setField("country", country);
  setField("lat", lat);
  setField("lng", lng);

  const requestHeaders = new Headers(req.headers);
  if (parts.length > 0) {
    requestHeaders.set(VISITOR_HEADER, parts.join(";"));
  }

  return NextResponse.next({ request: { headers: requestHeaders } });
}

export const config = {
  matcher: ["/api/:path*"],
};
