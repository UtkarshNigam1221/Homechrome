import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Ban, RotateCcw, Search } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { returnsApi } from '@/features/returns/api';
import { getErrorMessage } from '@/shared/api/client';
import { PageHeader } from '@/shared/components/layout';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
  Input,
  Modal,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '@/shared/components/ui';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency, formatPaiseExact } from '@/shared/utils/currency';

interface RefundAmountModalProps {
  isOpen: boolean;
  suggestedPaise: number;
  onClose: () => void;
  onSubmit: (paise: number) => void;
  isPending: boolean;
}

function RefundAmountForm({
  suggestedPaise,
  onClose,
  onSubmit,
  isPending,
}: Omit<RefundAmountModalProps, 'isOpen'>) {
  const [paise, setPaise] = useState<number>(suggestedPaise);

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (paise <= 0) return;
    onSubmit(paise);
  };

  return (
    <form onSubmit={submit} className="space-y-4">
      <Input
        type="number"
        step="1"
        min="1"
        label="Refund amount (paise)"
        value={paise}
        onChange={(e) => setPaise(Math.max(0, parseInt(e.target.value || '0', 10)))}
        required
      />
      <p className="text-xs text-gray-500">
        ≈ {formatPaiseExact(paise)}. Suggested: {formatPaiseExact(suggestedPaise)}.
      </p>
      <div className="flex justify-end gap-2 pt-4 border-t border-gray-200">
        <Button type="button" variant="secondary" onClick={onClose} disabled={isPending}>
          Cancel
        </Button>
        <Button type="submit" loading={isPending} disabled={paise <= 0}>
          Refund
        </Button>
      </div>
    </form>
  );
}

function RefundAmountModal({
  isOpen,
  suggestedPaise,
  onClose,
  onSubmit,
  isPending,
}: RefundAmountModalProps) {
  // Re-mount the inner form whenever the modal opens with a new suggested
  // amount so internal state is seeded fresh without a state-syncing effect.
  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Process refund" size="sm">
      <RefundAmountForm
        key={`${isOpen}-${suggestedPaise}`}
        suggestedPaise={suggestedPaise}
        onClose={onClose}
        onSubmit={onSubmit}
        isPending={isPending}
      />
    </Modal>
  );
}

export function ReturnsListPage() {
  const queryClient = useQueryClient();
  const [orderInput, setOrderInput] = useState('');
  const [orderId, setOrderId] = useState('');
  const [refundTarget, setRefundTarget] = useState<{ id: string; suggested: number } | null>(null);
  const [cancelTarget, setCancelTarget] = useState<string | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ['returns', 'by-order', orderId],
    queryFn: () => returnsApi.listForOrder(orderId),
    enabled: !!orderId,
  });

  const cancelMutation = useMutation({
    mutationFn: returnsApi.cancel,
    onSuccess: () => {
      toast.success('Return cancelled');
      if (orderId) {
        queryClient.invalidateQueries({ queryKey: ['returns', 'by-order', orderId] });
      }
    },
    onError: (e) => toast.error(getErrorMessage(e)),
  });

  const refundMutation = useMutation({
    mutationFn: ({ id, amount }: { id: string; amount: number }) =>
      returnsApi.processRefund(id, amount),
    onSuccess: () => {
      toast.success('Refund processed');
      if (orderId) {
        queryClient.invalidateQueries({ queryKey: ['returns', 'by-order', orderId] });
      }
    },
    onError: (e) => toast.error(getErrorMessage(e)),
  });

  const isCancelingRow = (id: string) =>
    cancelMutation.isPending && cancelMutation.variables === id;
  const isRefundingRow = (id: string) =>
    refundMutation.isPending && refundMutation.variables?.id === id;

  const returns = data ?? [];

  return (
    <div className="space-y-6">
      <PageHeader title="Returns" subtitle="Manage in-flight customer returns and refunds." />

      <Card padding="sm">
        <form
          className="flex gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            setOrderId(orderInput.trim());
          }}
        >
          <Input
            placeholder="Enter order ID to load returns"
            value={orderInput}
            onChange={(e) => setOrderInput(e.target.value)}
            leftIcon={<Search className="w-4 h-4" />}
            className="flex-1"
          />
          <Button type="submit" disabled={!orderInput.trim()}>
            Load
          </Button>
        </form>
      </Card>

      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Return ID</TableHead>
              <TableHead>Reverse AWB</TableHead>
              <TableHead>Items</TableHead>
              <TableHead>Refund amount</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {!orderId ? (
              <TableEmpty colSpan={6} message="Enter an order ID above to list returns" />
            ) : isLoading ? (
              <TableLoading rows={4} colSpan={6} />
            ) : error ? (
              <TableEmpty
                colSpan={6}
                message={getErrorMessage(error) || 'Failed to load returns'}
              />
            ) : returns.length === 0 ? (
              <TableEmpty colSpan={6} message="No returns for this order" />
            ) : (
              returns.map((rr) => {
                const suggested =
                  rr.refund_amount_paise > 0
                    ? rr.refund_amount_paise
                    : rr.items.reduce((s, it) => s + it.quantity * it.unit_paise, 0);
                return (
                  <TableRow key={rr.id}>
                    <TableCell>
                      <span className="font-mono text-xs">{rr.id}</span>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-sm">{rr.reverse_awb || '—'}</span>
                    </TableCell>
                    <TableCell>{rr.items?.length ?? 0}</TableCell>
                    <TableCell>
                      {rr.refund_amount_paise > 0 ? formatCurrency(rr.refund_amount_paise) : '—'}
                      {rr.refunded_at && (
                        <p className="text-xs text-gray-500">
                          on {format(new Date(rr.refunded_at), 'MMM d, yyyy')}
                        </p>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant={getStatusBadgeVariant(rr.status)}>{rr.status}</Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center justify-end gap-1">
                        {rr.status === 'RECEIVED' && (
                          <Button
                            variant="secondary"
                            size="sm"
                            leftIcon={<RotateCcw className="w-3.5 h-3.5" />}
                            onClick={() => setRefundTarget({ id: rr.id, suggested })}
                            loading={isRefundingRow(rr.id)}
                          >
                            Refund
                          </Button>
                        )}
                        {rr.status !== 'REFUNDED' && rr.status !== 'CANCELLED' && (
                          <Button
                            variant="ghost"
                            size="sm"
                            leftIcon={<Ban className="w-3.5 h-3.5" />}
                            onClick={() => setCancelTarget(rr.id)}
                            loading={isCancelingRow(rr.id)}
                            className="text-red-600 hover:text-red-700 hover:bg-red-50"
                          >
                            Cancel
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </Card>

      <RefundAmountModal
        isOpen={!!refundTarget}
        suggestedPaise={refundTarget?.suggested ?? 0}
        onClose={() => setRefundTarget(null)}
        isPending={refundMutation.isPending}
        onSubmit={(paise) => {
          if (!refundTarget) return;
          refundMutation.mutate(
            { id: refundTarget.id, amount: paise },
            { onSettled: () => setRefundTarget(null) }
          );
        }}
      />

      <ConfirmModal
        isOpen={!!cancelTarget}
        onClose={() => setCancelTarget(null)}
        onConfirm={() => {
          if (!cancelTarget) return;
          cancelMutation.mutate(cancelTarget, {
            onSettled: () => setCancelTarget(null),
          });
        }}
        title="Cancel return?"
        message="This will cancel the return request. The customer will not receive a refund. Continue?"
        confirmText="Cancel return"
        cancelText="Keep"
        loading={cancelMutation.isPending}
      />
    </div>
  );
}
