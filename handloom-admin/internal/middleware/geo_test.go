package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeoExtractor_PopulatesContext(t *testing.T) {
	var gotCity, gotCountry, gotDevice string
	var gotSrc, gotMed, gotCamp string
	var gotLat, gotLng float64
	var gotOK bool
	h := GeoExtractor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCity = GetCity(r.Context())
		gotCountry = GetCountry(r.Context())
		gotLat, gotLng, gotOK = GetLatLng(r.Context())
		gotDevice = GetDeviceType(r.Context())
		gotSrc, gotMed, gotCamp = GetUTM(r.Context())
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/x", nil)
	req.Header.Set(VisitorHeader,
		"city=Hyderabad;country=in;lat=17.3850;lng=78.4867;"+
			"device=Mobile;utm_source=Google;utm_medium=CPC;utm_campaign=Diwali_2026")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if gotCity != "hyderabad" {
		t.Errorf("city = %q, want hyderabad", gotCity)
	}
	if gotCountry != "IN" {
		t.Errorf("country = %q, want IN", gotCountry)
	}
	if !gotOK || gotLat != 17.3850 || gotLng != 78.4867 {
		t.Errorf("lat/lng = (%v, %v, ok=%v), want (17.385, 78.4867, ok=true)", gotLat, gotLng, gotOK)
	}
	if gotDevice != "mobile" {
		t.Errorf("device_type = %q, want mobile", gotDevice)
	}
	if gotSrc != "google" || gotMed != "cpc" || gotCamp != "diwali_2026" {
		t.Errorf("utm = (%q, %q, %q), want (google, cpc, diwali_2026)", gotSrc, gotMed, gotCamp)
	}
}

func TestGeoExtractor_URLDecodesValues(t *testing.T) {
	var gotCamp string
	h := GeoExtractor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _, gotCamp = GetUTM(r.Context())
	}))

	// Campaign with a `;` in it — URL-encoded by packer so it doesn't split
	// the visitor-header pairs.
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/x", nil)
	req.Header.Set(VisitorHeader, "utm_campaign=spring%3Bsale")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if gotCamp != "spring;sale" {
		t.Errorf("campaign = %q, want spring;sale", gotCamp)
	}
}

func TestGeoExtractor_DefaultsToUnknown(t *testing.T) {
	var gotCity, gotCountry, gotDevice string
	var gotSrc, gotMed, gotCamp string
	var gotOK bool
	h := GeoExtractor(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCity = GetCity(r.Context())
		gotCountry = GetCountry(r.Context())
		_, _, gotOK = GetLatLng(r.Context())
		gotDevice = GetDeviceType(r.Context())
		gotSrc, gotMed, gotCamp = GetUTM(r.Context())
	}))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/x", nil)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if gotCity != "unknown" {
		t.Errorf("city = %q, want unknown", gotCity)
	}
	if gotCountry != "unknown" {
		t.Errorf("country = %q, want unknown", gotCountry)
	}
	if gotOK {
		t.Errorf("lat/lng ok = true, want false (missing headers)")
	}
	if gotDevice != "unknown" {
		t.Errorf("device_type = %q, want unknown", gotDevice)
	}
	if gotSrc != "unknown" || gotMed != "unknown" || gotCamp != "unknown" {
		t.Errorf("utm = (%q, %q, %q), want all unknown", gotSrc, gotMed, gotCamp)
	}
}
