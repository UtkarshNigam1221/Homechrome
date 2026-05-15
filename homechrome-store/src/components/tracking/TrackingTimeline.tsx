import { formatDateTime } from '@/lib/utils';
import type { TrackingScan } from '@/types/shipping';

const STATUS_LABEL: Record<string, string> = {
  MANIFESTED: 'Manifested',
  PICKED_UP: 'Picked up',
  IN_TRANSIT: 'In transit',
  OUT_FOR_DELIVERY: 'Out for delivery',
  DELIVERED: 'Delivered',
  NDR: 'Delivery attempted',
  RTO_INITIATED: 'Return initiated',
  RTO_DELIVERED: 'Returned to sender',
  REVERSE_PICKED_UP: 'Return picked up',
  REVERSE_DELIVERED: 'Return delivered',
  PENDING: 'Order placed',
  CONFIRMED: 'Confirmed',
  PROCESSING: 'Processing',
  SHIPPED: 'Shipped',
  CANCELLED: 'Cancelled',
  RETURNED: 'Returned',
  REFUNDED: 'Refunded',
  UNKNOWN: 'Status update',
};

const STATUS_COLOR: Record<string, string> = {
  MANIFESTED: 'bg-blue-500',
  PICKED_UP: 'bg-blue-500',
  IN_TRANSIT: 'bg-indigo-500',
  OUT_FOR_DELIVERY: 'bg-amber-500',
  DELIVERED: 'bg-green-500',
  NDR: 'bg-red-500',
  RTO_INITIATED: 'bg-orange-500',
  RTO_DELIVERED: 'bg-orange-600',
  REVERSE_PICKED_UP: 'bg-purple-500',
  REVERSE_DELIVERED: 'bg-purple-600',
  PENDING: 'bg-yellow-500',
  CONFIRMED: 'bg-blue-500',
  PROCESSING: 'bg-indigo-500',
  SHIPPED: 'bg-purple-500',
  CANCELLED: 'bg-red-500',
  RETURNED: 'bg-orange-500',
  REFUNDED: 'bg-gray-500',
  UNKNOWN: 'bg-gray-400',
};

interface Props {
  scans: TrackingScan[];
  currentStatus?: string;
}

export default function TrackingTimeline({ scans, currentStatus }: Props) {
  if (!scans || scans.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No tracking updates available yet.
      </p>
    );
  }

  const currentIdx =
    currentStatus !== undefined
      ? scans.findIndex((s) => s.status === currentStatus)
      : -1;
  const highlightIdx = currentIdx === -1 ? 0 : currentIdx;

  return (
    <div className="relative">
      <div className="absolute left-3 top-2 bottom-2 w-0.5 bg-border" />

      <ol
        className="space-y-6"
        aria-label="Tracking history, most recent first"
      >
        {scans.map((scan, idx) => {
          const label = STATUS_LABEL[scan.status] ?? scan.status;
          const color = STATUS_COLOR[scan.status] ?? 'bg-gray-400';
          const isCurrent = idx === highlightIdx;

          return (
            <li
              key={`${scan.status}-${scan.timestamp}`}
              className="relative flex gap-4 pl-10"
            >
              <span
                aria-hidden="true"
                className={`absolute left-1 top-1 h-5 w-5 rounded-full border-2 ${
                  isCurrent ? `${color} border-transparent` : 'border-border bg-white'
                }`}
              >
                {isCurrent && (
                  <span className="flex h-full w-full items-center justify-center">
                    <span className="h-1.5 w-1.5 rounded-full bg-white" />
                  </span>
                )}
              </span>

              <div className="flex-1">
                <p className="font-medium text-foreground">{label}</p>
                <p className="text-xs text-muted-foreground">
                  {formatDateTime(scan.timestamp)}
                </p>
                {scan.location && (
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    {scan.location}
                  </p>
                )}
                {scan.description && (
                  <p className="mt-0.5 text-sm text-muted-foreground">
                    {scan.description}
                  </p>
                )}
              </div>
            </li>
          );
        })}
      </ol>
    </div>
  );
}
