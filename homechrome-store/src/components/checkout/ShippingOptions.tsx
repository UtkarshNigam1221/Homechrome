'use client';

import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { cn, formatPrice } from '@/lib/utils';
import { CourierOption } from '@/types';

interface ShippingOptionsProps {
  couriers: CourierOption[];
  selectedId: number | null;
  onSelect: (id: number) => void;
  loading?: boolean;
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
            className="h-20 animate-pulse rounded-lg border border-border bg-card"
          />
        ))}
      </div>
    );
  }

  if (couriers.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No shipping options available for this address.
      </p>
    );
  }

  return (
    <RadioGroup
      value={selectedId?.toString()}
      onValueChange={(val) => onSelect(Number(val))}
      aria-label="Shipping options"
    >
      {couriers.map((courier) => {
        const isSelected = selectedId === courier.id;
        return (
          <label
            key={courier.id}
            className={cn(
              'flex cursor-pointer items-center justify-between rounded-lg border p-4 transition-colors',
              isSelected
                ? 'border-primary bg-primary/5 ring-1 ring-primary'
                : 'border-border hover:border-primary/50',
            )}
          >
            <div className="flex items-center gap-3">
              <RadioGroupItem value={courier.id.toString()} />
              <div>
                <p className="font-medium text-foreground">{courier.name}</p>
                <p className="text-sm text-muted-foreground">
                  Estimated delivery in {courier.estimated_days}{' '}
                  {courier.estimated_days === 1 ? 'day' : 'days'}
                </p>
              </div>
            </div>
            <span className="font-semibold text-foreground">
              {courier.rate === 0 ? 'FREE' : formatPrice(courier.rate)}
            </span>
          </label>
        );
      })}
    </RadioGroup>
  );
}
