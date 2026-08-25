package service

import (
	"context"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// utmBaseURL is the production storefront. Campaign links are only ever handed to
// ad platforms, so there is no dev variant to switch on.
const utmBaseURL = "https://www.homechrome.in"

// utmValuePattern is what survives the storefront's capture path unchanged.
// Anything else is rejected rather than silently mangled: the storefront lowercases
// and truncates on capture (visitor-context.ts), so a value that needs mangling would
// produce a link whose params never match its own analytics rows.
var utmValuePattern = regexp.MustCompile(`^[a-z0-9._-]+$`)

// utmSlugPattern matches storefront category/product slugs.
var utmSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// UTMLinkService implements domain.UTMLinkService
type UTMLinkService struct {
	linkRepo domain.UTMLinkRepository
}

// NewUTMLinkService creates a new UTMLinkService
func NewUTMLinkService(linkRepo domain.UTMLinkRepository) *UTMLinkService {
	return &UTMLinkService{linkRepo: linkRepo}
}

// Create creates a new UTM link
func (s *UTMLinkService) Create(ctx context.Context, req domain.CreateUTMLinkRequest, createdBy string) (*domain.UTMLink, error) {
	link := &domain.UTMLink{
		ID:          "utm_" + uuid.New().String()[:8],
		Name:        strings.TrimSpace(req.Name),
		DestType:    req.DestType,
		DestSlug:    normalizeUTMValue(req.DestSlug),
		UTMSource:   normalizeUTMValue(req.UTMSource),
		UTMMedium:   normalizeUTMValue(req.UTMMedium),
		UTMCampaign: normalizeUTMValue(req.UTMCampaign),
	}
	link.CreatedBy = createdBy

	if err := s.finalize(link); err != nil {
		return nil, err
	}

	if err := s.linkRepo.Create(ctx, link); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Created UTM link", "utm_link_id", link.ID, "url", link.URL)
	return link, nil
}

// GetByID retrieves a UTM link by ID
func (s *UTMLinkService) GetByID(ctx context.Context, id string) (*domain.UTMLink, error) {
	return s.linkRepo.GetByID(ctx, id)
}

// Update updates an existing UTM link and rebuilds its URL
func (s *UTMLinkService) Update(ctx context.Context, id string, req domain.UpdateUTMLinkRequest, updatedBy string) (*domain.UTMLink, error) {
	link, err := s.linkRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		link.Name = strings.TrimSpace(*req.Name)
	}
	if req.DestType != nil {
		link.DestType = *req.DestType
	}
	if req.DestSlug != nil {
		link.DestSlug = normalizeUTMValue(*req.DestSlug)
	}
	if req.UTMSource != nil {
		link.UTMSource = normalizeUTMValue(*req.UTMSource)
	}
	if req.UTMMedium != nil {
		link.UTMMedium = normalizeUTMValue(*req.UTMMedium)
	}
	if req.UTMCampaign != nil {
		link.UTMCampaign = normalizeUTMValue(*req.UTMCampaign)
	}

	if err := s.finalize(link); err != nil {
		return nil, err
	}

	link.UpdatedBy = updatedBy

	if err := s.linkRepo.Update(ctx, link); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Updated UTM link", "utm_link_id", id, "url", link.URL)
	return link, nil
}

// Delete deletes a UTM link by ID
func (s *UTMLinkService) Delete(ctx context.Context, id string) error {
	if err := s.linkRepo.Delete(ctx, id); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Deleted UTM link", "utm_link_id", id)
	return nil
}

// List retrieves UTM links
func (s *UTMLinkService) List(ctx context.Context, req domain.ListUTMLinksRequest) (*domain.ListUTMLinksResponse, error) {
	return s.linkRepo.List(ctx, req)
}

// finalize validates the normalized fields and stamps the built URL. Shared by
// Create and Update so an edit can never leave a link whose URL disagrees with
// its own parts.
func (s *UTMLinkService) finalize(link *domain.UTMLink) error {
	if link.Name == "" {
		return errors.Validation("Name is required")
	}

	for label, value := range map[string]string{
		"utm_source":   link.UTMSource,
		"utm_medium":   link.UTMMedium,
		"utm_campaign": link.UTMCampaign,
	} {
		if err := validateUTMValue(label, value); err != nil {
			return err
		}
	}

	if err := validateUTMDestination(link.DestType, link.DestSlug); err != nil {
		return err
	}
	if link.DestType == domain.UTMDestHome {
		// Kept clear so an edit from CATEGORY to HOME can't leave a stale slug behind.
		link.DestSlug = ""
	}

	link.URL = buildUTMURL(link)
	return nil
}

// normalizeUTMValue applies the same shaping the storefront applies on capture:
// trim, then lowercase. Length is validated rather than truncated.
func normalizeUTMValue(input string) string {
	return strings.ToLower(strings.TrimSpace(input))
}

func validateUTMValue(label, value string) error {
	if value == "" {
		return errors.Validation(label + " is required")
	}
	if len(value) > domain.MaxUTMValueLen {
		return errors.Newf(errors.ErrCodeValidation, "%s must be at most %d characters; longer values are truncated on capture and would not match analytics",
			label, domain.MaxUTMValueLen)
	}
	if !utmValuePattern.MatchString(value) {
		return errors.Newf(errors.ErrCodeValidation, "%s may only contain lowercase letters, digits, dot, underscore or hyphen", label)
	}
	return nil
}

func validateUTMDestination(destType domain.UTMDestType, slug string) error {
	switch destType {
	case domain.UTMDestHome:
		return nil
	case domain.UTMDestCategory, domain.UTMDestProduct:
		if slug == "" {
			return errors.Validation("Destination slug is required for a category or product link")
		}
		if !utmSlugPattern.MatchString(slug) {
			return errors.Validation("Destination slug may only contain lowercase letters, digits and hyphens")
		}
		return nil
	default:
		return errors.Newf(errors.ErrCodeValidation, "Unknown destination type %q", destType)
	}
}

// buildUTMURL assembles base + path + the three params. Params are sorted by
// url.Values.Encode, so the same inputs always produce a byte-identical link.
func buildUTMURL(link *domain.UTMLink) string {
	path := ""
	switch link.DestType {
	case domain.UTMDestCategory:
		path = "/c/" + link.DestSlug
	case domain.UTMDestProduct:
		path = "/p/" + link.DestSlug
	case domain.UTMDestHome:
		path = "/"
	}

	q := url.Values{}
	q.Set("utm_source", link.UTMSource)
	q.Set("utm_medium", link.UTMMedium)
	q.Set("utm_campaign", link.UTMCampaign)

	return utmBaseURL + path + "?" + q.Encode()
}
