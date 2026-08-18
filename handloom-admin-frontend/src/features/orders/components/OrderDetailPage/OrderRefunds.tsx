import { useMutation, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { RefreshCw, RotateCcw } from 'lucide-react';
import toast from 'react-hot-toast';

import { ordersApi } from '@/features/orders/api';
import { getErrorMessage } from '@/shared/api/client';
import { Badge, Button, Card } from '@/shared/components/ui';
import { formatCurrencyExact } from '@/shared/utils/currency';

import type { Refund, RefundStatus } from '../../types';
import { REFUND_REASON_LABELS } from '../../types';

export interface OrderRefundsProps {
  orderId: string;
  refunds: Refund[];
  isLoading: boolean;
}

// A refund raised by an admin names them. One settled straight from a webhook has
// no actor, and inventing one would be worse than showing none.
function actorName(refund: Refund): string {
  return refund.created_by_name ?? refund.created_by;
}

const STATUS_VARIANT: Record<RefundStatus, 'warning' | 'success' | 'danger'> = {
  PENDING: 'warning',
  COMPLETED: 'success',
  FAILED: 'danger',
};

export function OrderRefunds({ orderId, refunds, isLoading }: OrderRefundsProps) {
  const queryClient = useQueryClient();

  const recheckMutation = useMutation({
    mutationFn: (refundId: string) => ordersApi.recheckRefund(orderId, refundId),
    onSuccess: (refund) => {
      toast.success(
        refund.status === 'PENDING'
          ? 'Still pending at the provider'
          : `Refund is now ${refund.status.toLowerCase()}`
      );
      void queryClient.invalidateQueries({ queryKey: ['order-refunds', orderId] });
      void queryClient.invalidateQueries({ queryKey: ['order', orderId] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  if (isLoading || refunds.length === 0) return null;

  return (
    <Card>
      <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
        <RotateCcw className="w-5 h-5" />
        Refunds ({refunds.length})
      </h2>
      <div className="divide-y divide-gray-200">
        {refunds.map((refund) => (
          <div key={refund.id} className="py-4 flex items-start justify-between gap-4">
            <div className="min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <span className="font-mono font-medium text-gray-900">
                  {formatCurrencyExact(refund.amount)}
                </span>
                <Badge variant={STATUS_VARIANT[refund.status]}>{refund.status}</Badge>
                <span className="text-sm text-gray-500">
                  {REFUND_REASON_LABELS[refund.reason] ?? refund.reason}
                </span>
              </div>
              <p className="text-sm text-gray-500 mt-1">
                {refund.items
                  .map(
                    (item) =>
                      `${item.quantity} × ${item.product_name} (${item.restock ? 'back to stock' : 'written off'})`
                  )
                  .join(', ')}
              </p>
              <p className="text-xs text-gray-400 mt-1">
                {format(new Date(refund.initiated_at), 'd MMM yyyy, h:mm a')}
                {/* Money left the account on someone's say-so; the record has to name them. */}
                {actorName(refund) && ` by ${actorName(refund)}`}
                {refund.completed_at &&
                  ` · settled ${format(new Date(refund.completed_at), 'd MMM, h:mm a')}`}
              </p>
              {refund.status === 'FAILED' && refund.error_code && (
                <p className="text-xs text-red-600 mt-1">
                  {refund.error_code}
                  {refund.detailed_error_code && ` — ${refund.detailed_error_code}`}
                </p>
              )}
            </div>

            {/* Only a pending refund has anything to learn from the provider. */}
            {refund.status === 'PENDING' && (
              <Button
                variant="secondary"
                size="sm"
                leftIcon={<RefreshCw className="w-3.5 h-3.5" />}
                loading={recheckMutation.isPending && recheckMutation.variables === refund.id}
                onClick={() => recheckMutation.mutate(refund.id)}
              >
                Re-check
              </Button>
            )}
          </div>
        ))}
      </div>
    </Card>
  );
}
