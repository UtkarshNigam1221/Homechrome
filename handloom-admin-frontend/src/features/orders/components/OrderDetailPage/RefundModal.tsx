import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, Truck } from 'lucide-react';
import { useMemo, useState } from 'react';
import toast from 'react-hot-toast';

import { ordersApi } from '@/features/orders/api';
import type { RefundClaims } from '@/features/orders/lib/refundAmount';
import { previewRefund, unrefundedQuantity } from '@/features/orders/lib/refundAmount';
import { getErrorMessage } from '@/shared/api/client';
import { Button, Input, Modal, Select } from '@/shared/components/ui';
import { formatCurrencyExact } from '@/shared/utils/currency';

import type { Order, RefundReason } from '../../types';
import { REFUND_REASON_LABELS } from '../../types';

export interface RefundModalProps {
  isOpen: boolean;
  onClose: () => void;
  order: Order;
  /** What the order's refunds already account for, settled or in flight. */
  priorRefunded: number;
  /** Units of each line those refunds already claim. */
  claims: RefundClaims;
  /** A refund still awaiting the provider. Its units are not yet off the order. */
  hasPendingRefund: boolean;
}

const REASON_OPTIONS = (Object.keys(REFUND_REASON_LABELS) as RefundReason[]).map((value) => ({
  value,
  label: REFUND_REASON_LABELS[value],
}));

// Dispatch is the line where a refund stops moving stock: CommitStock has already
// consumed the reservation, and RETURNED owns restocking from there. Letting a
// refund restock as well would count the same goods back twice.
function isDispatched(order: Order): boolean {
  return ['SHIPPED', 'DELIVERED', 'RETURNED'].includes(order.status);
}

export function RefundModal({
  isOpen,
  onClose,
  order,
  priorRefunded,
  claims,
  hasPendingRefund,
}: RefundModalProps) {
  const queryClient = useQueryClient();

  const [quantities, setQuantities] = useState<Record<string, number>>({});
  const [restock, setRestock] = useState<Record<string, boolean>>({});
  const [reason, setReason] = useState<RefundReason | ''>('');

  // The modal stays mounted between openings, so nothing resets on its own —
  // without this a second refund opens holding the first one's selection.
  // Adjusted during render rather than in an effect, which would re-render the
  // stale selection first.
  const [wasOpen, setWasOpen] = useState(isOpen);
  if (isOpen !== wasOpen) {
    setWasOpen(isOpen);
    if (isOpen) {
      setQuantities({});
      setRestock({});
      setReason('');
    }
  }

  const items = useMemo(() => order.items ?? [], [order.items]);
  const refundable = useMemo(
    () => items.filter((item) => unrefundedQuantity(item, claims) > 0),
    [items, claims]
  );
  const dispatched = isDispatched(order);

  const preview = useMemo(
    () => previewRefund(order, quantities, priorRefunded, claims),
    [order, quantities, priorRefunded, claims]
  );

  // From the priced lines, not the raw input: a line that dropped out while the
  // modal was open must not count towards an enabled button.
  const selectedUnits = preview.lines.reduce((sum, line) => sum + line.quantity, 0);
  const remainingAfter = order.total_amount - priorRefunded - preview.total;

  const refundMutation = useMutation({
    mutationFn: () =>
      ordersApi.createRefund(order.id, {
        reason: reason as RefundReason,
        // The quantity that was priced, not the raw input — they differ if a
        // pending refund settles while the modal is open.
        items: preview.lines.map((line) => ({
          order_item_id: line.orderItemId,
          quantity: line.quantity,
          restock: restock[line.orderItemId] ?? false,
        })),
      }),
    onSuccess: (refund) => {
      toast.success(`Refund of ${formatCurrencyExact(refund.amount)} sent to the provider`);
      void queryClient.invalidateQueries({ queryKey: ['order', order.id] });
      void queryClient.invalidateQueries({ queryKey: ['order-refunds', order.id] });
      void queryClient.invalidateQueries({ queryKey: ['orders'] });
      // A refund moves stock as well as money, so the inventory views go stale too.
      for (const key of ['products', 'products-inventory', 'inventory', 'low-stock']) {
        void queryClient.invalidateQueries({ queryKey: [key] });
      }
      onClose();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const setQuantity = (itemId: string, value: number, max: number) => {
    setQuantities((current) => ({ ...current, [itemId]: Math.min(Math.max(value, 0), max) }));
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Refund order #${order.order_number || order.id.slice(0, 8)}`}
      description={`Paid ${formatCurrencyExact(order.total_amount)} · ${formatCurrencyExact(priorRefunded)} already refunded`}
      size="xl"
    >
      <div className="space-y-5">
        {hasPendingRefund && (
          <div className="flex gap-2 rounded-lg bg-amber-50 border border-amber-200 p-3 text-sm text-amber-900">
            <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
            <p>
              A refund on this order is still pending at the provider. Its units are not counted
              below until it settles — check it before refunding the same ones twice.
            </p>
          </div>
        )}

        {dispatched && (
          <div className="flex gap-2 rounded-lg bg-gray-50 border border-gray-200 p-3 text-sm text-gray-700">
            <Truck className="w-4 h-4 mt-0.5 shrink-0" />
            <p>
              This order has already been dispatched, so the refund moves money only. To bring the
              goods back into stock, mark the order returned.
            </p>
          </div>
        )}

        {refundable.length === 0 ? (
          <p className="text-sm text-gray-500 py-6 text-center">
            Every line on this order has already been refunded.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-left text-xs uppercase tracking-wide text-gray-500 border-b border-gray-200">
                  <th className="py-2 pr-4 font-medium">Item</th>
                  <th className="py-2 pr-4 font-medium text-right">Unit price</th>
                  <th className="py-2 pr-4 font-medium text-right">Left</th>
                  <th className="py-2 pr-4 font-medium">Refund</th>
                  {!dispatched && <th className="py-2 pr-4 font-medium">Stock</th>}
                  <th className="py-2 font-medium text-right">Amount</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {refundable.map((item) => {
                  const left = unrefundedQuantity(item, claims);
                  const quantity = quantities[item.id] ?? 0;
                  const line = preview.lines.find((entry) => entry.orderItemId === item.id);

                  return (
                    <tr key={item.id}>
                      <td className="py-3 pr-4">
                        <p className="font-medium text-gray-900">{item.product_name}</p>
                        <p className="text-xs text-gray-500">{item.product_sku}</p>
                      </td>
                      <td className="py-3 pr-4 text-right font-mono text-gray-700">
                        {formatCurrencyExact(item.unit_price)}
                      </td>
                      <td className="py-3 pr-4 text-right text-gray-700">
                        {left}
                        {item.refunded_quantity > 0 && (
                          <span className="text-xs text-gray-400"> of {item.quantity}</span>
                        )}
                      </td>
                      <td className="py-3 pr-4">
                        <div className="flex items-center gap-2">
                          <Input
                            type="number"
                            min={0}
                            max={left}
                            value={quantity}
                            aria-label={`Refund quantity for ${item.product_name}`}
                            onChange={(event) =>
                              setQuantity(item.id, Number(event.target.value), left)
                            }
                            className="w-20"
                          />
                          <button
                            type="button"
                            className="text-xs text-primary-600 hover:underline whitespace-nowrap"
                            onClick={() => setQuantity(item.id, left, left)}
                          >
                            All {left}
                          </button>
                        </div>
                      </td>
                      {!dispatched && (
                        <td className="py-3 pr-4">
                          <Select
                            aria-label={`Stock handling for ${item.product_name}`}
                            value={restock[item.id] ? 'restock' : 'write_off'}
                            disabled={quantity === 0}
                            onChange={(event) =>
                              setRestock((current) => ({
                                ...current,
                                [item.id]: event.target.value === 'restock',
                              }))
                            }
                            options={[
                              // Written off by default: "cannot serve this" usually
                              // means the goods are not there to sell again.
                              { value: 'write_off', label: 'Write off' },
                              { value: 'restock', label: 'Back to stock' },
                            ]}
                            className="w-36"
                          />
                        </td>
                      )}
                      <td className="py-3 text-right font-mono text-gray-900">
                        {line ? formatCurrencyExact(line.amount) : '—'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {refundable.length > 0 && (
          <>
            <Select
              label="Reason"
              required
              placeholder="Pick a reason"
              value={reason}
              onChange={(event) => setReason(event.target.value as RefundReason)}
              options={REASON_OPTIONS}
              hint="Anything that needs explaining belongs in an order note."
            />

            <div className="rounded-lg bg-gray-50 border border-gray-200 p-4 space-y-2">
              {/* The terms, not just the total: a line refunding less than its
                  own value should not look like an error. */}
              <div className="flex justify-between text-sm text-gray-600">
                <span>
                  {selectedUnits} unit{selectedUnits === 1 ? '' : 's'} across {preview.lines.length}{' '}
                  line{preview.lines.length === 1 ? '' : 's'}
                </span>
                <span className="font-mono">
                  {formatCurrencyExact(preview.breakdown.lineValue)}
                </span>
              </div>
              {preview.breakdown.discount > 0 && (
                <div className="flex justify-between text-sm text-gray-600">
                  <span>Less their share of the discount</span>
                  <span className="font-mono">
                    −{formatCurrencyExact(preview.breakdown.discount)}
                  </span>
                </div>
              )}
              {preview.breakdown.tax > 0 && (
                <div className="flex justify-between text-sm text-gray-600">
                  <span>Their share of the tax</span>
                  <span className="font-mono">{formatCurrencyExact(preview.breakdown.tax)}</span>
                </div>
              )}
              <div className="flex justify-between text-sm text-gray-600">
                <span>
                  {preview.isFinal ? 'Shipping' : 'Shipping (kept until the order clears)'}
                </span>
                <span className="font-mono">{formatCurrencyExact(preview.breakdown.shipping)}</span>
              </div>
              <div className="flex justify-between text-sm font-medium text-gray-900 border-t border-gray-200 pt-2">
                <span>Refund total</span>
                <span className="font-mono">{formatCurrencyExact(preview.total)}</span>
              </div>
              {preview.isFinal ? (
                <p className="text-sm text-emerald-700">
                  This clears the order — the shipping comes back with it.
                </p>
              ) : (
                <p className="text-sm text-gray-500">
                  {formatCurrencyExact(remainingAfter)} of the order would stay live.
                </p>
              )}
              <p className="text-xs text-gray-400">
                The server derives the final figure. This is what it should come to.
              </p>
            </div>
          </>
        )}

        <div className="flex justify-end gap-3">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="danger"
            disabled={selectedUnits === 0 || reason === ''}
            loading={refundMutation.isPending}
            onClick={() => refundMutation.mutate()}
          >
            Refund {formatCurrencyExact(preview.total)}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
