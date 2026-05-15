import type { ReturnStatus } from '@/types';

const LABEL: Record<ReturnStatus, string> = {
  REQUESTED: 'Return requested',
  PICKED_UP: 'Pickup completed',
  IN_TRANSIT: 'Returning to warehouse',
  RECEIVED: 'Return received',
  REFUNDED: 'Refunded',
  CANCELLED: 'Return cancelled',
};

const COLOR: Record<ReturnStatus, string> = {
  REQUESTED: 'bg-amber-100 text-amber-800',
  PICKED_UP: 'bg-blue-100 text-blue-800',
  IN_TRANSIT: 'bg-blue-200 text-blue-900',
  RECEIVED: 'bg-purple-100 text-purple-800',
  REFUNDED: 'bg-green-100 text-green-800',
  CANCELLED: 'bg-gray-200 text-gray-700',
};

interface Props {
  status: ReturnStatus;
}

export default function ReturnStatusBadge({ status }: Props) {
  return (
    <span
      className={`inline-flex rounded-full px-3 py-1 text-xs font-medium ${
        COLOR[status] ?? 'bg-gray-100 text-gray-700'
      }`}
    >
      {LABEL[status] ?? status}
    </span>
  );
}
