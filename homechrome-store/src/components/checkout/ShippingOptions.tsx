'use client';

import { CourierOption } from '@/types';

interface ShippingOptionsProps {
  couriers: CourierOption[];
  selectedId: number | null;
  onSelect: (id: number) => void;
  loading?: boolean;
}

function formatPrice(paise: number): string {
  return `₹${(paise / 100).toLocaleString('en-IN')}`;
}

export default function ShippingOptions({
  couriers,
  selectedId,
  onSelect,
  loading = false,
}: ShippingOptionsProps) {
  if (loading) {
    return (
      <div className="space-y-3">
        {[1, 2].map((i) => (
          <div
            key={i}
            className="h-20 animate-pulse rounded-lg border border-border bg-white"
          />
        ))}
      </div>
    );
  }

  if (couriers.length === 0) {
    return (
      <p className="text-sm text-muted">
        No shipping options available for this address.
      </p>
    );
  }

  return (
    <div className="space-y-3">
      {couriers.map((courier) => (
        <button
          key={courier.id}
          type="button"
          onClick={() => onSelect(courier.id)}
          className={`w-full rounded-lg border p-4 text-left transition-colors ${
            selectedId === courier.id
              ? 'border-primary bg-primary/5 ring-1 ring-primary'
              : 'border-border bg-white hover:border-primary/50'
          }`}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div
                className={`flex h-5 w-5 items-center justify-center rounded-full border-2 ${
                  selectedId === courier.id
                    ? 'border-primary'
                    : 'border-muted'
                }`}
              >
                {selectedId === courier.id && (
                  <div className="h-2.5 w-2.5 rounded-full bg-primary" />
                )}
              </div>
              <div>
                <p className="font-medium text-foreground">{courier.name}</p>
                <p className="text-sm text-muted">
                  Estimated delivery in {courier.estimated_days}{' '}
                  {courier.estimated_days === 1 ? 'day' : 'days'}
                </p>
              </div>
            </div>
            <span className="font-semibold text-foreground">
              {courier.rate === 0 ? 'FREE' : formatPrice(courier.rate)}
            </span>
          </div>
        </button>
      ))}
    </div>
  );
}
