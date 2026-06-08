package middleware

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/handloom/admin/pkg/geo"
)

type geoCityKey struct{}
type geoCountryKey struct{}
type geoLatKey struct{}
type geoLngKey struct{}
type deviceTypeKey struct{}
type utmSourceKey struct{}
type utmMediumKey struct{}
type utmCampaignKey struct{}

// MaxUTMLen caps the length of UTM label values. Same defensive bound as
// MaxCityNameLen — UTM values come from arbitrary marketing URLs and can
// run very long if uncapped.
const MaxUTMLen = 32

// VisitorHeader is the single request header carrying all visitor-attribution
// fields (city, country, lat, lng, device, utm_*). Storefront axios assembles
// it browser-side; Next.js middleware merges in CloudFront viewer geo bits
// before forwarding to backend. Single header keeps the CloudFront
// OriginRequestPolicy AllowList small (10-header default cap).
//
// Format: key=value pairs joined by `;`. Values are URL-encoded so they can
// safely carry `;`, `=`, or `,`. Empty / unknown keys are simply omitted.
//
//	X-Hc-Visitor: city=hyderabad;country=IN;lat=17.385;lng=78.4867;device=mobile;utm_source=google;utm_medium=cpc;utm_campaign=diwali_2026
const VisitorHeader = "X-Hc-Visitor"

// GeoExtractor reads the X-Hc-Visitor header, parses its key=value pairs,
// normalises each, and stores them in the request context. Getters
// (GetCity, GetCountry, GetLatLng, GetDeviceType, GetUTM) return the
// stowed values with "unknown" / zero defaults when fields are absent.
func GeoExtractor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fields := parseVisitorHeader(r.Header.Get(VisitorHeader))

		city := geo.NormalizeCity(fields["city"])
		country := geo.NormalizeCountry(fields["country"])

		lat, _ := strconv.ParseFloat(fields["lat"], 64)
		lng, _ := strconv.ParseFloat(fields["lng"], 64)

		device := geo.NormalizeDevice(fields["device"])
		utmSource := truncUTM(fields["utm_source"])
		utmMedium := truncUTM(fields["utm_medium"])
		utmCampaign := truncUTM(fields["utm_campaign"])

		ctx := r.Context()
		ctx = context.WithValue(ctx, geoCityKey{}, city)
		ctx = context.WithValue(ctx, geoCountryKey{}, country)
		ctx = context.WithValue(ctx, geoLatKey{}, lat)
		ctx = context.WithValue(ctx, geoLngKey{}, lng)
		ctx = context.WithValue(ctx, deviceTypeKey{}, device)
		ctx = context.WithValue(ctx, utmSourceKey{}, utmSource)
		ctx = context.WithValue(ctx, utmMediumKey{}, utmMedium)
		ctx = context.WithValue(ctx, utmCampaignKey{}, utmCampaign)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// parseVisitorHeader splits "key1=v1;key2=v2;..." into a map. Values are
// URL-decoded so packing-side can safely encode `;` / `=` / `,` inside
// campaign names. Unknown / malformed pairs are silently dropped.
func parseVisitorHeader(raw string) map[string]string {
	out := make(map[string]string, 8)
	if raw == "" {
		return out
	}
	for _, pair := range strings.Split(raw, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		eq := strings.IndexByte(pair, '=')
		if eq <= 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(pair[:eq]))
		val, err := url.QueryUnescape(strings.TrimSpace(pair[eq+1:]))
		if err != nil {
			continue
		}
		out[key] = val
	}
	return out
}

func truncUTM(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	if s == "" {
		return "unknown"
	}
	if len(s) > MaxUTMLen {
		s = s[:MaxUTMLen]
	}
	return s
}

// GetCity returns the geo-city string from context, or "unknown".
func GetCity(ctx context.Context) string {
	if v, ok := ctx.Value(geoCityKey{}).(string); ok {
		return v
	}
	return "unknown"
}

// GetCountry returns the ISO-2 country code from context, or "unknown".
func GetCountry(ctx context.Context) string {
	if v, ok := ctx.Value(geoCountryKey{}).(string); ok {
		return v
	}
	return "unknown"
}

// GetLatLng returns the viewer latitude / longitude pair from context.
// Returns (0, 0, false) when either value is missing or zero so callers
// can skip writes that would seed bogus centroids.
func GetLatLng(ctx context.Context) (lat, lng float64, ok bool) {
	lat, _ = ctx.Value(geoLatKey{}).(float64)
	lng, _ = ctx.Value(geoLngKey{}).(float64)
	return lat, lng, lat != 0 && lng != 0
}

// GetDeviceType returns the device type from context, or "unknown".
func GetDeviceType(ctx context.Context) string {
	if v, ok := ctx.Value(deviceTypeKey{}).(string); ok {
		return v
	}
	return "unknown"
}

// GetUTM returns the (utm_source, utm_medium, utm_campaign) tuple from
// context, each defaulting to "unknown" if absent.
func GetUTM(ctx context.Context) (source, medium, campaign string) {
	s, _ := ctx.Value(utmSourceKey{}).(string)
	m, _ := ctx.Value(utmMediumKey{}).(string)
	c, _ := ctx.Value(utmCampaignKey{}).(string)
	if s == "" {
		s = "unknown"
	}
	if m == "" {
		m = "unknown"
	}
	if c == "" {
		c = "unknown"
	}
	return s, m, c
}
