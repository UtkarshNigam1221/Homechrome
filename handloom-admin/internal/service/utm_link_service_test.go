package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

// fakeUTMLinkRepo records what the service handed it. A hand-rolled fake rather than
// a mock: every test here asserts on the built URL, not on call expectations.
type fakeUTMLinkRepo struct {
	saved    *domain.UTMLink
	existing *domain.UTMLink
}

func (f *fakeUTMLinkRepo) Create(_ context.Context, link *domain.UTMLink) error {
	f.saved = link
	return nil
}

func (f *fakeUTMLinkRepo) GetByID(_ context.Context, _ string) (*domain.UTMLink, error) {
	return f.existing, nil
}

func (f *fakeUTMLinkRepo) Update(_ context.Context, link *domain.UTMLink) error {
	f.saved = link
	return nil
}

func (f *fakeUTMLinkRepo) Delete(_ context.Context, _ string) error { return nil }

func (f *fakeUTMLinkRepo) List(_ context.Context, _ domain.ListUTMLinksRequest) (*domain.ListUTMLinksResponse, error) {
	return &domain.ListUTMLinksResponse{}, nil
}

func TestUTMLinkService_Create_BuildsURL(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.CreateUTMLinkRequest
		wantURL string
	}{
		{
			name: "home",
			req: domain.CreateUTMLinkRequest{
				Name: "Diwali homepage", DestType: domain.UTMDestHome,
				UTMSource: "google", UTMMedium: "cpc", UTMCampaign: "diwali_2026",
			},
			wantURL: "https://www.homechrome.in/?utm_campaign=diwali_2026&utm_medium=cpc&utm_source=google",
		},
		{
			name: "category",
			req: domain.CreateUTMLinkRequest{
				Name: "Sarees insta", DestType: domain.UTMDestCategory, DestSlug: "cotton-sarees",
				UTMSource: "instagram", UTMMedium: "social", UTMCampaign: "spring",
			},
			wantURL: "https://www.homechrome.in/c/cotton-sarees?utm_campaign=spring&utm_medium=social&utm_source=instagram",
		},
		{
			name: "product",
			req: domain.CreateUTMLinkRequest{
				Name: "Hero product", DestType: domain.UTMDestProduct, DestSlug: "kanjivaram-silk-01",
				UTMSource: "whatsapp", UTMMedium: "referral", UTMCampaign: "launch",
			},
			wantURL: "https://www.homechrome.in/p/kanjivaram-silk-01?utm_campaign=launch&utm_medium=referral&utm_source=whatsapp",
		},
		{
			// The storefront lowercases on capture, so the saved link must already be
			// lowercase or its params never match the analytics rows they produce.
			name: "normalizes case and whitespace",
			req: domain.CreateUTMLinkRequest{
				Name: "Mixed case", DestType: domain.UTMDestHome,
				UTMSource: "  Google  ", UTMMedium: "CPC", UTMCampaign: " Diwali_2026 ",
			},
			wantURL: "https://www.homechrome.in/?utm_campaign=diwali_2026&utm_medium=cpc&utm_source=google",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUTMLinkRepo{}
			svc := NewUTMLinkService(repo)

			link, err := svc.Create(context.Background(), tt.req, "tester")
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, link.URL)
			assert.Equal(t, tt.wantURL, repo.saved.URL, "stored URL must match the returned one")
		})
	}
}

func TestUTMLinkService_Create_Rejects(t *testing.T) {
	tests := []struct {
		name string
		req  domain.CreateUTMLinkRequest
	}{
		{
			// 33 chars: the storefront would truncate to 32 on capture, so this link
			// could never match its own analytics rows.
			name: "campaign over 32 chars",
			req: domain.CreateUTMLinkRequest{
				Name: "Too long", DestType: domain.UTMDestHome,
				UTMSource: "google", UTMMedium: "cpc", UTMCampaign: "aaaaaaaaaabbbbbbbbbbccccccccccddd",
			},
		},
		{
			name: "source with a space",
			req: domain.CreateUTMLinkRequest{
				Name: "Bad source", DestType: domain.UTMDestHome,
				UTMSource: "google ads", UTMMedium: "cpc", UTMCampaign: "diwali",
			},
		},
		{
			name: "category without slug",
			req: domain.CreateUTMLinkRequest{
				Name: "No slug", DestType: domain.UTMDestCategory,
				UTMSource: "google", UTMMedium: "cpc", UTMCampaign: "diwali",
			},
		},
		{
			name: "empty medium",
			req: domain.CreateUTMLinkRequest{
				Name: "No medium", DestType: domain.UTMDestHome,
				UTMSource: "google", UTMMedium: "", UTMCampaign: "diwali",
			},
		},
		{
			name: "blank name",
			req: domain.CreateUTMLinkRequest{
				Name: "   ", DestType: domain.UTMDestHome,
				UTMSource: "google", UTMMedium: "cpc", UTMCampaign: "diwali",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeUTMLinkRepo{}
			svc := NewUTMLinkService(repo)

			_, err := svc.Create(context.Background(), tt.req, "tester")
			require.Error(t, err)
			assert.Nil(t, repo.saved, "nothing may be persisted when validation fails")
		})
	}
}

// Switching a category link to HOME must drop the slug, or the rebuilt URL would
// keep pointing at /c/<slug>.
func TestUTMLinkService_Update_ClearsSlugOnHome(t *testing.T) {
	repo := &fakeUTMLinkRepo{existing: &domain.UTMLink{
		ID: "utm_abc123", Name: "Sarees insta",
		DestType: domain.UTMDestCategory, DestSlug: "cotton-sarees",
		UTMSource: "instagram", UTMMedium: "social", UTMCampaign: "spring",
	}}
	svc := NewUTMLinkService(repo)

	home := domain.UTMDestHome
	link, err := svc.Update(context.Background(), "utm_abc123", domain.UpdateUTMLinkRequest{DestType: &home}, "tester")
	require.NoError(t, err)

	assert.Empty(t, link.DestSlug)
	assert.Equal(t, "https://www.homechrome.in/?utm_campaign=spring&utm_medium=social&utm_source=instagram", link.URL)
}
