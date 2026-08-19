import { useQuery } from '@tanstack/react-query';
import { AlertTriangle } from 'lucide-react';
import { Link } from 'react-router-dom';

import { inventoryApi } from '@/features/inventory/api';
import { Card } from '@/shared/components/ui';

function formatWhen(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

/**
 * Stock held against orders that never dispatched and never cancelled.
 *
 * Renders nothing when there is nothing wrong — this is a report worth noticing,
 * so it must not become furniture people learn to scroll past.
 */
export function ReconciliationPanel() {
  const { data } = useQuery({
    queryKey: ['inventory-reconciliation'],
    queryFn: () => inventoryApi.getReconciliation(),
    // The backing query is a ledger scan, not a page load.
    staleTime: 5 * 60 * 1000,
  });

  if (!data || data.order_count === 0) return null;

  return (
    <Card className="border-amber-200 bg-amber-50">
      <div className="flex items-start gap-3">
        <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
        <div className="min-w-0 flex-1">
          <h2 className="font-semibold text-amber-900">
            {data.stranded_units} unit{data.stranded_units === 1 ? '' : 's'} reserved against{' '}
            {data.order_count} order{data.order_count === 1 ? '' : 's'} that never settled
          </h2>
          <p className="mt-1 text-sm text-amber-800">
            These are held out of stock but were never dispatched or released. Open each order and
            either ship it or cancel it — the runbook covers what to do when neither applies.
          </p>
          <div className="mt-3 space-y-1">
            {data.reservations.map((row) => (
              <div
                key={`${row.product_id}-${row.order_id}`}
                className="flex flex-wrap items-baseline gap-x-2 text-sm"
              >
                <Link
                  to={`/orders/${row.order_id}`}
                  className="font-mono text-blue-700 hover:underline"
                >
                  {row.order_id}
                </Link>
                <span className="text-amber-900">
                  {row.quantity} × {row.product_name}
                </span>
                <span className="text-xs text-amber-700">
                  {row.sku} · held since {formatWhen(row.reserved_at)}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </Card>
  );
}
