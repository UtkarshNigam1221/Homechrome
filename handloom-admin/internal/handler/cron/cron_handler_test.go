package cron

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
)

func TestRateRefreshHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rt := mocks.NewMockRateTableService(ctrl)
	rt.EXPECT().Refresh(gomock.Any()).Return(&domain.RefreshResult{RowsUpdated: 24, RowsSkipped: 1}, nil)
	h := NewRateRefreshHandler(rt)
	require.NoError(t, h.Handle(context.Background()))
}

func TestRateRefreshHandler_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rt := mocks.NewMockRateTableService(ctrl)
	rt.EXPECT().Refresh(gomock.Any()).Return(nil, errors.New("api down"))
	h := NewRateRefreshHandler(rt)
	assert.Error(t, h.Handle(context.Background()))
}

func TestCODRemittanceHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	c := mocks.NewMockCODReconciliationService(ctrl)
	c.EXPECT().RunDailyPull(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&domain.PullResult{RemittancesProcessed: 2, EntriesMatched: 5, EntriesUnmatched: 1}, nil)
	h := NewCODRemittanceHandler(c)
	require.NoError(t, h.Handle(context.Background()))
}

func TestPickupBatchHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mocks.NewMockManifestService(ctrl)
	m.EXPECT().RunDailyBatch(gomock.Any(), gomock.Any()).
		Return(&domain.BatchResult{ManifestID: "MAN-1", ShipmentCount: 12}, nil)
	h := NewPickupBatchHandler(m)
	require.NoError(t, h.Handle(context.Background()))
}

func TestCODRemittanceHandler_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	c := mocks.NewMockCODReconciliationService(ctrl)
	c.EXPECT().RunDailyPull(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("delhivery down"))
	h := NewCODRemittanceHandler(c)
	assert.Error(t, h.Handle(context.Background()))
}

func TestPickupBatchHandler_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mocks.NewMockManifestService(ctrl)
	m.EXPECT().RunDailyBatch(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("ddb throttle"))
	h := NewPickupBatchHandler(m)
	assert.Error(t, h.Handle(context.Background()))
}
