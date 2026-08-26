import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import {
  ArrowLeft,
  Ban,
  CreditCard,
  Edit,
  MapPin,
  MessageSquare,
  Package,
  RotateCcw,
  Truck,
} from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { useNavigate, useParams } from 'react-router-dom';

import { addressFullName } from '@/features/customers/lib/displayName';
import { ordersApi } from '@/features/orders/api';
import { claimedByLiveRefunds } from '@/features/orders/lib/refundClaims';
import { getErrorMessage } from '@/shared/api/client';
import { Badge, Button, Card, Input, Modal, Select } from '@/shared/components/ui';
import { useAuthStore } from '@/shared/stores/authStore';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency, formatCurrencyExact } from '@/shared/utils/currency';

import type { OrderStatus, ProviderPaymentStatus } from '../../types';
import { ALLOWED_TRANSITIONS, FORWARD_STATUSES } from '../../types';
import { OrderNotes } from './OrderNotes';
import { OrderRefunds } from './OrderRefunds';
import { OrderTimeline } from './OrderTimeline';
import { RefundModal } from './RefundModal';

export function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isAdmin = useAuthStore((state) => state.user?.role === 'ADMIN');

  const [showStatusModal, setShowStatusModal] = useState(false);
  const [showTrackingModal, setShowTrackingModal] = useState(false);
  const [showNoteModal, setShowNoteModal] = useState(false);
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [showRefundModal, setShowRefundModal] = useState(false);
  const [newStatus, setNewStatus] = useState<OrderStatus | ''>('');
  const [trackingNumber, setTrackingNumber] = useState('');
  const [carrier, setCarrier] = useState('');
  const [trackingUrl, setTrackingUrl] = useState('');
  const [noteText, setNoteText] = useState('');
  const [cancelReason, setCancelReason] = useState('');
  const [noteIsInternal, setNoteIsInternal] = useState(true);

  // Fetch order
  const {
    data: order,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['order', id],
    queryFn: () => ordersApi.get(id ?? ''),
    enabled: !!id,
  });

  const { data: refunds = [], isLoading: refundsLoading } = useQuery({
    queryKey: ['order-refunds', id],
    queryFn: () => ordersApi.listRefunds(id ?? ''),
    enabled: !!id,
  });

  // What the order's refunds already account for, settled or in flight, reconciled
  // against the payment's own figure — the same reading the server takes.
  const { claims: refundClaims, amount: priorRefunded } = claimedByLiveRefunds(
    refunds,
    order?.refunded_amount ?? 0
  );
  const hasPendingRefund = refunds.some((refund) => refund.status === 'PENDING');

  // Update status mutation
  const updateStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      ordersApi.updateStatus(id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      toast.success('Order status updated');
      setShowStatusModal(false);
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  // Update tracking mutation
  const updateTrackingMutation = useMutation({
    mutationFn: ({
      id,
      tracking_number,
      carrier,
      tracking_url,
    }: {
      id: string;
      tracking_number: string;
      carrier: string;
      tracking_url: string;
    }) => ordersApi.updateTracking(id, tracking_number, carrier, tracking_url),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      toast.success('Tracking information updated');
      setShowTrackingModal(false);
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  // Add note mutation
  const addNoteMutation = useMutation({
    mutationFn: ({ id, note, is_internal }: { id: string; note: string; is_internal: boolean }) =>
      ordersApi.addNote(id, note, is_internal),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      toast.success('Note added');
      setShowNoteModal(false);
      setNoteText('');
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  // Cancel order mutation
  const cancelMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => ordersApi.cancel(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      toast.success('Order cancelled');
      setShowCancelModal(false);
      setCancelReason('');
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  // Payment status check
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [paymentStatus, setPaymentStatus] = useState<ProviderPaymentStatus | null>(null);
  const checkPaymentMutation = useMutation({
    mutationFn: (id: string) => ordersApi.checkPaymentStatus(id),
    onSuccess: (data) => {
      // The re-check reconciles the payment server-side, so the order on screen
      // is stale the moment it returns.
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      setPaymentStatus(data);
      setShowPaymentModal(true);
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="h-8 bg-gray-200 rounded w-64 animate-pulse" />
        <div className="h-96 bg-gray-200 rounded animate-pulse" />
      </div>
    );
  }

  if (error || !order) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500">Order not found</p>
        <Button className="mt-4" onClick={() => navigate('/orders')}>
          Back to Orders
        </Button>
      </div>
    );
  }

  // Up to dispatch, matching CancelOrder's own guard. This is the only route to a
  // cancellation now that the status dropdown does not offer one.
  const canCancel = ['PENDING', 'CONFIRMED', 'PROCESSING'].includes(order.status);
  // A failed payment is the one case the page can be sure the backend will refuse.
  // PENDING is not: it may be an admin order with no payment at all, which passes.
  const paymentFailed = order.payment_status === 'FAILED';
  const nextStatuses = ALLOWED_TRANSITIONS[order.status].filter(
    (s) => !paymentFailed || !FORWARD_STATUSES.includes(s)
  );

  // Refunding is gated on the money having arrived, and on the role. The backend
  // enforces both; hiding the button just stops the pointless attempt.
  const canRefund =
    isAdmin &&
    ['PAID', 'SUCCESS', 'PARTIALLY_REFUNDED'].includes(order.payment_status) &&
    order.items?.some(
      (item) => item.quantity > Math.max(item.refunded_quantity ?? 0, refundClaims[item.id] ?? 0)
    );

  // Mirrors the backend's `http_url` validation so a bad paste is caught before
  // the request. The scheme check matters: this URL becomes a customer-facing
  // link, and a javascript: URL would otherwise sail through.
  const trackingUrlError =
    trackingUrl.trim() && !/^https?:\/\/\S+$/i.test(trackingUrl.trim())
      ? 'Enter a URL starting with http:// or https://'
      : undefined;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" onClick={() => navigate('/orders')}>
            <ArrowLeft className="w-4 h-4" />
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              Order #{order.order_number || order.id.slice(0, 8)}
            </h1>
            <p className="text-sm text-gray-500">
              Placed on {format(new Date(order.created_at), 'MMMM d, yyyy h:mm a')}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={getStatusBadgeVariant(order.status)}>{order.status}</Badge>
          <Badge variant={getStatusBadgeVariant(order.payment_status)}>
            {order.payment_status}
          </Badge>
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex flex-wrap gap-2">
        <Button
          variant="secondary"
          leftIcon={<Edit className="w-4 h-4" />}
          // Stays clickable on a failed payment even with nothing to offer: the modal
          // is where the reason lives, and a dead grey button explains nothing.
          disabled={nextStatuses.length === 0 && !paymentFailed}
          onClick={() => {
            // Seed with the first legal next status — the current one is not an
            // option, since the backend rejects a no-op transition.
            setNewStatus(nextStatuses[0] ?? '');
            setShowStatusModal(true);
          }}
        >
          Update Status
        </Button>
        <Button
          variant="secondary"
          leftIcon={<Truck className="w-4 h-4" />}
          onClick={() => {
            setTrackingNumber(order.tracking_number || '');
            setCarrier(order.shipping_carrier || '');
            setTrackingUrl(order.tracking_url || '');
            setShowTrackingModal(true);
          }}
        >
          Update Tracking
        </Button>
        <Button
          variant="secondary"
          leftIcon={<CreditCard className="w-4 h-4" />}
          onClick={() => checkPaymentMutation.mutate(order.id)}
          loading={checkPaymentMutation.isPending}
        >
          Check Payment
        </Button>
        <Button
          variant="secondary"
          leftIcon={<MessageSquare className="w-4 h-4" />}
          onClick={() => setShowNoteModal(true)}
        >
          Add Note
        </Button>
        {canRefund && (
          <Button
            variant="secondary"
            leftIcon={<RotateCcw className="w-4 h-4" />}
            onClick={() => setShowRefundModal(true)}
          >
            Refund
          </Button>
        )}
        {canCancel && (
          <Button
            variant="danger"
            leftIcon={<Ban className="w-4 h-4" />}
            onClick={() => setShowCancelModal(true)}
          >
            Cancel Order
          </Button>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Order Items */}
          <Card>
            <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <Package className="w-5 h-5" />
              Order Items ({order.items?.length || 0})
            </h2>
            <div className="divide-y divide-gray-200">
              {order.items?.map((item, index) => (
                <div key={item.id || index} className="py-4 flex justify-between items-start">
                  <div className="flex gap-4">
                    <div className="w-16 h-16 bg-gray-100 rounded-lg flex items-center justify-center">
                      <Package className="w-6 h-6 text-gray-400" />
                    </div>
                    <div>
                      <p className="font-medium">{item.product_name}</p>
                      <p className="text-sm text-gray-500">SKU: {item.product_sku}</p>
                      <p className="text-sm text-gray-500">
                        Qty: {item.quantity}
                        {item.refunded_quantity > 0 && (
                          <span className="text-amber-700">
                            {' '}
                            · {item.refunded_quantity} refunded
                          </span>
                        )}
                      </p>
                      {item.custom_dimensions && (
                        <p className="text-sm text-gray-500">
                          Size: {item.custom_dimensions.length} x {item.custom_dimensions.width}{' '}
                          {item.custom_dimensions.unit}
                        </p>
                      )}
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-medium">{formatCurrency(item.total_price)}</p>
                    <p className="text-sm text-gray-500">{formatCurrency(item.unit_price)} each</p>
                  </div>
                </div>
              ))}
            </div>
          </Card>

          {/* Order Summary */}
          <Card>
            <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <CreditCard className="w-5 h-5" />
              Payment Summary
            </h2>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span className="text-gray-600">Subtotal</span>
                <span>{formatCurrency(order.subtotal)}</span>
              </div>
              {order.discount_amount > 0 && (
                <div className="flex justify-between text-green-600">
                  <span>Discount {order.coupon_code && `(${order.coupon_code})`}</span>
                  <span>-{formatCurrency(order.discount_amount)}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-gray-600">Shipping</span>
                <span>{formatCurrency(order.shipping_amount)}</span>
              </div>
              <div className="border-t pt-2 flex justify-between font-semibold text-lg">
                <span>Total</span>
                <span>{formatCurrency(order.total_amount)}</span>
              </div>
              {order.tax_amount > 0 && (
                <div className="flex justify-between text-xs text-gray-400">
                  <span>Of which GST</span>
                  <span>{formatCurrency(order.tax_amount)}</span>
                </div>
              )}
              {priorRefunded > 0 && (
                <div className="flex justify-between text-amber-700">
                  <span>Refunded</span>
                  <span className="font-mono">−{formatCurrencyExact(priorRefunded)}</span>
                </div>
              )}
            </div>
          </Card>

          <OrderRefunds
            orderId={order.id}
            refunds={refunds}
            isLoading={refundsLoading}
            canRecheck={isAdmin}
          />

          {/* Notes */}
          {order.internal_notes && order.internal_notes.length > 0 && (
            <OrderNotes notes={order.internal_notes} />
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Customer Info */}
          <Card>
            <h2 className="text-lg font-semibold mb-4">Customer</h2>
            <div className="space-y-3">
              <div>
                <p className="font-medium">{order.customer_name}</p>
                <p className="text-sm text-gray-500">{order.customer_email}</p>
              </div>
              <Button
                variant="secondary"
                size="sm"
                disabled={!order.customer_id}
                onClick={() => navigate(`/customers?id=${order.customer_id}`)}
              >
                View Customer
              </Button>
            </div>
          </Card>

          {/* Shipping Address */}
          <Card>
            <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <MapPin className="w-5 h-5" />
              Shipping Address
            </h2>
            <div className="text-sm">
              <p className="font-medium">{addressFullName(order.shipping_address)}</p>
              <p className="text-gray-600">{order.shipping_address?.address_line1}</p>
              {order.shipping_address?.address_line2 && (
                <p className="text-gray-600">{order.shipping_address.address_line2}</p>
              )}
              <p className="text-gray-600">
                {order.shipping_address?.city}, {order.shipping_address?.state}{' '}
                {order.shipping_address?.postal_code}
              </p>
              <p className="text-gray-600">{order.shipping_address?.country}</p>
              {order.shipping_address?.phone && (
                <p className="text-gray-600 mt-2">Phone: {order.shipping_address.phone}</p>
              )}
            </div>
          </Card>

          {/* Tracking Info */}
          {order.tracking_number && (
            <Card>
              <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
                <Truck className="w-5 h-5" />
                Tracking
              </h2>
              <div className="space-y-2">
                <div>
                  <p className="text-sm text-gray-500">Carrier</p>
                  <p className="font-medium">{order.shipping_carrier || 'N/A'}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Tracking Number</p>
                  <p className="font-mono text-sm">{order.tracking_number}</p>
                </div>
                {order.tracking_url && (
                  <div>
                    <p className="text-sm text-gray-500">Tracking URL</p>
                    <a
                      href={order.tracking_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-sm text-primary-600 hover:underline break-all"
                    >
                      {order.tracking_url}
                    </a>
                  </div>
                )}
              </div>
            </Card>
          )}

          {/* Timeline */}
          <OrderTimeline createdAt={order.created_at} updatedAt={order.updated_at} />
        </div>
      </div>

      {canRefund && (
        <RefundModal
          isOpen={showRefundModal}
          onClose={() => setShowRefundModal(false)}
          order={order}
          priorRefunded={priorRefunded}
          claims={refundClaims}
          hasPendingRefund={hasPendingRefund}
        />
      )}

      {/* Update Status Modal */}
      <Modal
        isOpen={showStatusModal}
        onClose={() => setShowStatusModal(false)}
        title="Update Order Status"
        size="sm"
      >
        <div className="space-y-4">
          {nextStatuses.length === 0 ? (
            <p className="text-sm text-gray-600">
              {paymentFailed
                ? 'This order cannot move forward while its payment has failed. Cancel it instead, or collect the payment first.'
                : `An order that is ${order.status.toLowerCase()} cannot move to another status.`}
            </p>
          ) : (
            <Select
              label="New Status"
              options={nextStatuses.map((s) => ({ value: s, label: s }))}
              value={newStatus}
              onChange={(e) => setNewStatus(e.target.value as OrderStatus)}
            />
          )}
          <div className="flex justify-end gap-3 pt-4">
            <Button variant="secondary" onClick={() => setShowStatusModal(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => updateStatusMutation.mutate({ id: order.id, status: newStatus })}
              loading={updateStatusMutation.isPending}
              disabled={!newStatus || updateStatusMutation.isPending}
            >
              Update Status
            </Button>
          </div>
        </div>
      </Modal>

      {/* Update Tracking Modal */}
      <Modal
        isOpen={showTrackingModal}
        onClose={() => setShowTrackingModal(false)}
        title="Update Tracking Information"
        size="sm"
      >
        <div className="space-y-4">
          <Input
            label="Carrier"
            placeholder="e.g., FedEx, DHL, Blue Dart"
            value={carrier}
            onChange={(e) => setCarrier(e.target.value)}
          />
          <Input
            label="Tracking Number"
            placeholder="Enter tracking number"
            value={trackingNumber}
            onChange={(e) => setTrackingNumber(e.target.value)}
          />
          <Input
            label="Tracking URL (optional)"
            placeholder="https://carrier.example/track/123"
            value={trackingUrl}
            onChange={(e) => setTrackingUrl(e.target.value)}
            error={trackingUrlError}
            hint="Shown to the customer as a “Track on courier website” link."
          />
          <div className="flex justify-end gap-3 pt-4">
            <Button variant="secondary" onClick={() => setShowTrackingModal(false)}>
              Cancel
            </Button>
            <Button
              onClick={() =>
                updateTrackingMutation.mutate({
                  id: order.id,
                  tracking_number: trackingNumber,
                  carrier,
                  tracking_url: trackingUrl.trim(),
                })
              }
              loading={updateTrackingMutation.isPending}
              disabled={!trackingNumber || !!trackingUrlError}
            >
              Update Tracking
            </Button>
          </div>
        </div>
      </Modal>

      {/* Add Note Modal */}
      <Modal
        isOpen={showNoteModal}
        onClose={() => setShowNoteModal(false)}
        title="Add Order Note"
        size="sm"
      >
        <div className="space-y-4">
          <div>
            <label className="label">Note</label>
            <textarea
              className="input min-h-[100px]"
              placeholder="Enter note..."
              value={noteText}
              onChange={(e) => setNoteText(e.target.value)}
            />
          </div>
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="internal_note"
              checked={noteIsInternal}
              onChange={(e) => setNoteIsInternal(e.target.checked)}
              className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            />
            <label htmlFor="internal_note" className="text-sm text-gray-700">
              Internal note (only visible to staff)
            </label>
          </div>
          {!noteIsInternal && (
            <p className="text-sm text-yellow-700 bg-yellow-50 border border-yellow-200 rounded-lg p-3">
              This note will be shown to the customer on their order page.
            </p>
          )}
          <div className="flex justify-end gap-3 pt-4">
            <Button variant="secondary" onClick={() => setShowNoteModal(false)}>
              Cancel
            </Button>
            <Button
              onClick={() =>
                addNoteMutation.mutate({
                  id: order.id,
                  note: noteText,
                  is_internal: noteIsInternal,
                })
              }
              loading={addNoteMutation.isPending}
              disabled={!noteText.trim()}
            >
              Add Note
            </Button>
          </div>
        </div>
      </Modal>

      {/* Payment Status Modal */}
      <Modal
        isOpen={showPaymentModal}
        onClose={() => setShowPaymentModal(false)}
        title="Payment Provider Status"
        size="sm"
      >
        {paymentStatus && (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <p className="text-gray-500">Provider State</p>
                <Badge variant={getStatusBadgeVariant(paymentStatus.provider_state)}>
                  {paymentStatus.provider_state}
                </Badge>
              </div>
              <div>
                <p className="text-gray-500">Local Status</p>
                <Badge variant={getStatusBadgeVariant(paymentStatus.local_status)}>
                  {paymentStatus.local_status}
                </Badge>
              </div>
              <div>
                <p className="text-gray-500">Merchant Txn ID</p>
                <p className="font-mono text-xs break-all">{paymentStatus.merchant_txn_id}</p>
              </div>
              <div>
                <p className="text-gray-500">Provider Order ID</p>
                <p className="font-mono text-xs break-all">{paymentStatus.provider_order_id}</p>
              </div>
              <div>
                <p className="text-gray-500">Amount</p>
                <p className="font-medium">{formatCurrency(paymentStatus.amount)}</p>
              </div>
              {paymentStatus.payment_mode && (
                <div>
                  <p className="text-gray-500">Payment Mode</p>
                  <p className="font-medium">{paymentStatus.payment_mode}</p>
                </div>
              )}
              {paymentStatus.transaction_id && (
                <div className="col-span-2">
                  <p className="text-gray-500">Transaction ID</p>
                  <p className="font-mono text-xs break-all">{paymentStatus.transaction_id}</p>
                </div>
              )}
            </div>
            <div className="flex justify-end pt-4">
              <Button variant="secondary" onClick={() => setShowPaymentModal(false)}>
                Close
              </Button>
            </div>
          </div>
        )}
      </Modal>

      {/* Cancel Order Modal. Asks for a reason because the backend requires one —
          and because this is the only cancellation path that records it. */}
      <Modal
        isOpen={showCancelModal}
        onClose={() => setShowCancelModal(false)}
        title="Cancel Order"
        size="sm"
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-600">
            Cancel order #{order.order_number || order.id.slice(0, 8)}? This releases the reserved
            stock and cannot be undone.
          </p>
          <Input
            label="Reason"
            value={cancelReason}
            onChange={(e) => setCancelReason(e.target.value)}
            placeholder="e.g. payment failed at the gateway"
          />
          <div className="flex justify-end gap-3">
            <Button variant="secondary" onClick={() => setShowCancelModal(false)}>
              Keep Order
            </Button>
            <Button
              variant="danger"
              onClick={() => cancelMutation.mutate({ id: order.id, reason: cancelReason.trim() })}
              loading={cancelMutation.isPending}
              disabled={!cancelReason.trim()}
            >
              Cancel Order
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
