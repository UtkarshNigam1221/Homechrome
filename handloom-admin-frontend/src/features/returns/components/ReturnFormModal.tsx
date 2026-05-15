import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import toast from 'react-hot-toast';

import type { Order, OrderItem } from '@/features/orders/types';
import { returnsApi } from '@/features/returns/api';
import type { ReturnItem } from '@/features/returns/types';
import { getErrorMessage } from '@/shared/api/client';
import { Button, Input, Modal } from '@/shared/components/ui';
import { formatCurrency } from '@/shared/utils/currency';

interface ReturnFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  order: Order | null;
}

interface PickerRow {
  product_id: string;
  sku: string;
  name: string;
  available: number;
  selected: number;
  unit_paise: number;
}

function rowsFromOrder(items: OrderItem[] | undefined): PickerRow[] {
  return (items ?? []).map((it) => ({
    product_id: it.product_id,
    sku: it.product_sku,
    name: it.product_name,
    available: it.quantity,
    selected: 0,
    unit_paise: it.unit_price,
  }));
}

function ReturnFormInner({ onClose, order }: { onClose: () => void; order: Order }) {
  const queryClient = useQueryClient();
  const [rows, setRows] = useState<PickerRow[]>(() => rowsFromOrder(order.items));
  const [reason, setReason] = useState('');

  const total = useMemo(() => rows.reduce((sum, r) => sum + r.selected * r.unit_paise, 0), [rows]);

  const mutation = useMutation({
    mutationFn: ({ orderId, items }: { orderId: string; items: ReturnItem[] }) =>
      returnsApi.createReturn(orderId, { items, reason }),
    onSuccess: () => {
      toast.success('Return request created');
      queryClient.invalidateQueries({ queryKey: ['order', order.id] });
      queryClient.invalidateQueries({ queryKey: ['returns', 'by-order', order.id] });
      onClose();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const handleQty = (idx: number, qty: number) => {
    if (Number.isNaN(qty) || !Number.isFinite(qty)) qty = 0;
    setRows((prev) =>
      prev.map((r, i) =>
        i === idx ? { ...r, selected: Math.max(0, Math.min(r.available, qty)) } : r
      )
    );
  };

  const handleSubmit = () => {
    const items: ReturnItem[] = rows
      .filter((r) => r.selected > 0)
      .map((r) => ({
        product_id: r.product_id,
        sku: r.sku,
        quantity: r.selected,
        unit_paise: r.unit_paise,
      }));
    if (items.length === 0) {
      toast.error('Select at least one item');
      return;
    }
    if (!reason.trim()) {
      toast.error('Reason is required');
      return;
    }
    mutation.mutate({ orderId: order.id, items });
  };

  return (
    <div className="space-y-4">
      <div className="text-sm text-gray-500">
        Order <span className="font-mono">{order.order_number || order.id}</span> ·{' '}
        {order.customer_name}
      </div>

      <div className="border border-gray-200 rounded-lg divide-y divide-gray-100">
        {rows.length === 0 ? (
          <p className="p-4 text-sm text-gray-500">No items on this order</p>
        ) : (
          rows.map((row, idx) => (
            <div key={`${row.product_id}-${row.sku}`} className="flex items-center gap-3 p-3">
              <div className="flex-1 min-w-0">
                <p className="font-medium truncate">{row.name}</p>
                <p className="text-xs text-gray-500">
                  SKU {row.sku} · {formatCurrency(row.unit_paise)} · avail {row.available}
                </p>
              </div>
              <div className="w-24">
                <Input
                  type="number"
                  min={0}
                  max={row.available}
                  value={row.selected}
                  onChange={(e) => handleQty(idx, Number(e.target.value))}
                  aria-label={`Return quantity for ${row.name}`}
                />
              </div>
            </div>
          ))
        )}
      </div>

      <div>
        <label className="label" htmlFor="return-reason">
          Reason
        </label>
        <textarea
          id="return-reason"
          className="input min-h-[80px]"
          placeholder="Why is the customer returning?"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
      </div>

      <div className="flex items-center justify-between pt-3 border-t border-gray-200">
        <div className="text-sm text-gray-500">
          Estimated refund · <span className="font-medium">{formatCurrency(total)}</span>
        </div>
        <div className="flex gap-3">
          <Button variant="secondary" onClick={onClose} disabled={mutation.isPending}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} loading={mutation.isPending}>
            Create Return
          </Button>
        </div>
      </div>
    </div>
  );
}

export function ReturnFormModal({ isOpen, onClose, order }: ReturnFormModalProps) {
  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Initiate Return" size="lg">
      {order ? (
        // Re-mount the inner component on each open / order change so internal
        // form state is freshly seeded from the order's items without an effect.
        <ReturnFormInner key={`${order.id}-${isOpen}`} order={order} onClose={onClose} />
      ) : (
        <p className="text-gray-500">No order selected</p>
      )}
    </Modal>
  );
}
