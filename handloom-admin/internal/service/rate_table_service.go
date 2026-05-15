package service

import (
	"context"
	"fmt"
	"time"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/courier"
	"github.com/handloom/admin/pkg/errors"
)

// RateTableService refreshes the carrier rate matrix and resolves shipping
// charges for a given pincode + weight + payment mode. Manual overrides in
// the existing rate table are preserved on refresh.
type RateTableService struct {
	rateRepo    domain.ShippingRateRepository
	pincodeRepo domain.PincodeRepository
	courier     courier.Gateway
}

// NewRateTableService creates a RateTableService.
func NewRateTableService(rateRepo domain.ShippingRateRepository, pincodeRepo domain.PincodeRepository, gw courier.Gateway) *RateTableService {
	return &RateTableService{rateRepo: rateRepo, pincodeRepo: pincodeRepo, courier: gw}
}

// Refresh pulls the latest rate matrix from the carrier and upserts rows into
// the rate table. Rows previously flagged as manual overrides are skipped.
func (s *RateTableService) Refresh(ctx context.Context) (*domain.RefreshResult, error) {
	rows, err := s.courier.FetchRateMatrix(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch rate matrix")
	}
	existing, err := s.rateRepo.ListAll(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to list existing rates")
	}
	manualOverrides := map[string]bool{}
	for _, r := range existing {
		if r.Source == domain.RateSourceManualOverride {
			manualOverrides[r.Zone+"#"+fmt.Sprint(r.WeightSlabGrams)] = true
		}
	}
	res := &domain.RefreshResult{}
	toUpsert := make([]*domain.ShippingRate, 0, len(rows))
	for _, row := range rows {
		key := row.Zone + "#" + fmt.Sprint(row.WeightSlabGrams)
		if manualOverrides[key] {
			res.RowsSkipped++
			continue
		}
		toUpsert = append(toUpsert, &domain.ShippingRate{
			Zone:            row.Zone,
			WeightSlabGrams: row.WeightSlabGrams,
			PrepaidPaise:    row.PrepaidPaise,
			CODPaise:        row.CODPaise,
			RTOPaise:        row.RTOPaise,
			RefreshedAt:     time.Now().UTC(),
			Source:          domain.RateSourceDelhiveryAPI,
		})
	}
	if len(toUpsert) > 0 {
		if err := s.rateRepo.BatchUpsert(ctx, toUpsert); err != nil {
			return res, errors.Wrap(err, "Failed to batch upsert rates")
		}
	}
	res.RowsUpdated = len(toUpsert)
	return res, nil
}

// Lookup returns the shipping charge (in paise) for the given pincode + weight
// + payment mode. The pincode must already be cached in the zone table — call
// ShippingService.CheckServiceability first to populate it.
func (s *RateTableService) Lookup(ctx context.Context, pincode string, weightGrams int, mode domain.PaymentMode) (int64, error) {
	pz, err := s.pincodeRepo.Get(ctx, pincode)
	if err != nil {
		return 0, errors.Wrap(err, "Pincode not in zone cache; run CheckServiceability first")
	}
	slab := ceilingSlab(weightGrams)
	rate, err := s.rateRepo.Get(ctx, pz.Zone, slab)
	if err != nil {
		return 0, errors.Wrap(err, "Rate not found for zone/slab")
	}
	if mode == domain.PaymentModeCOD {
		return rate.CODPaise, nil
	}
	return rate.PrepaidPaise, nil
}

// ceilingSlab returns the smallest standard weight slab (in grams) that
// covers the given weight.
func ceilingSlab(grams int) int {
	for _, slab := range []int{500, 1000, 2000, 5000, 10000} {
		if grams <= slab {
			return slab
		}
	}
	return 10000
}

// Ensure interface compliance.
var _ domain.RateTableService = (*RateTableService)(nil)
