import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { shippingApi } from '@/features/shipping/api';
import type { ShippingRate } from '@/features/shipping/types';

import { RatesPage } from '../RatesPage';

vi.mock('@/features/shipping/api', () => ({
  shippingApi: {
    listRates: vi.fn(),
    triggerRateRefresh: vi.fn(),
  },
}));

const mockedApi = shippingApi as unknown as {
  listRates: ReturnType<typeof vi.fn>;
  triggerRateRefresh: ReturnType<typeof vi.fn>;
};

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <RatesPage />
    </QueryClientProvider>
  );
}

describe('RatesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders the rate matrix returned by the API', async () => {
    const rates: ShippingRate[] = [
      {
        zone: 'A',
        weight_slab_grams: 500,
        prepaid_paise: 5500,
        cod_paise: 6500,
        rto_paise: 4000,
        refreshed_at: '2025-01-01T00:00:00Z',
        source: 'delhivery_api',
      },
      {
        zone: 'B',
        weight_slab_grams: 1000,
        prepaid_paise: 9500,
        cod_paise: 11000,
        rto_paise: 7000,
        refreshed_at: '2025-01-01T00:00:00Z',
        source: 'manual_override',
      },
    ];
    mockedApi.listRates.mockResolvedValue(rates);

    renderPage();

    expect(screen.getByText('Shipping Rates')).toBeInTheDocument();
    await waitFor(() => expect(mockedApi.listRates).toHaveBeenCalled());

    expect(await screen.findByText('A')).toBeInTheDocument();
    expect(screen.getByText('B')).toBeInTheDocument();
    expect(screen.getByText('500 g')).toBeInTheDocument();
    expect(screen.getByText('1000 g')).toBeInTheDocument();
    // One row is API-sourced and one is manual-override
    expect(screen.getByText('API')).toBeInTheDocument();
    expect(screen.getByText('Manual')).toBeInTheDocument();
  });

  it('shows the empty state when no rates are present', async () => {
    mockedApi.listRates.mockResolvedValue([]);

    renderPage();

    expect(await screen.findByText('No shipping rates loaded yet')).toBeInTheDocument();
  });
});
