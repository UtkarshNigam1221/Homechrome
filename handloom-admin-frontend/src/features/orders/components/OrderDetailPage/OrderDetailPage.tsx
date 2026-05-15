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

import { ordersApi } from '@/features/orders/api';
import { ReturnFormModal } from '@/features/returns';
import { PriorityBadge, type ShipmentPriority } from '@/features/shipping';
import { getErrorMessage } from '@/shared/api/client';
import { Badge, Button, Card, ConfirmModal, Input, Modal, Select } from '@/shared/components/ui';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency } from '@/shared/utils/currency';

import type { OrderStatus, ProviderPaymentStatus } from '../../types';
import { ORDER_STATUSES } from '../../types';
import { PriorityToggle } from '../PriorityToggle';
import { OrderNotes } from './OrderNotes';
import { OrderTimeline } from './OrderTimeline';

export function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [showStatusModal, setShowStatusModal] = useState(false);
  const [showTrackingModal, setShowTrackingModal] = useState(false);
  const [showNoteModal, setShowNoteModal] = useState(false);
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [newStatus, setNewStatus] = useState<OrderStatus | ''>('');
  const [trackingNumber, setTrackingNumber] = useState('');
  const [carrier, setCarrier] = useState('');
  const [noteText, setNoteText] = useState('');
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
    }: {
      id: string;
      tracking_number: string;
      carrier: string;
    }) => ordersApi.updateTracking(id, tracking_number, carrier),
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
    mutationFn: (id: string) => ordersApi.cancel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      toast.success('Order cancelled');
      setShowCancelModal(false);
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  // Payment status check
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [paymentStatus, setPaymentStatus] = useState<ProviderPaymentStatus | null>(null);
  const checkPaymentMutation = useMutation({
    mutationFn: (id: string) => ordersApi.checkPaymentStatus(id),
    onSuccess: (data) => {
      setPaymentStatus(data);
      setShowPaymentModal(true);
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  // Shipment + returns
  const [priority, setPriority] = useState<ShipmentPriority>(
    () => order?.shipment?.priority ?? 'NORMAL'
  );
  const [showReturnModal, setShowReturnModal] = useState(false);
  const createShipmentMutation = useMutation({
    mutationFn: ({ id, p }: { id: string; p: ShipmentPriority }) => ordersApi.createShipment(id, p),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', id] });
      toast.success('Shipment created');
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

  const canCancel = ['PENDING', 'CONFIRMED'].includes(order.status);
  const canCreateShipment = ['CONFIRMED', 'PROCESSING'].includes(order.status) && !order.shipment;
  const canInitiateReturn = order.status === 'DELIVERED';

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
          onClick={() => {
            setNewStatus(order.status);
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
        {canInitiateReturn && (
          <Button
            variant="secondary"
            leftIcon={<RotateCcw className="w-4 h-4" />}
            onClick={() => setShowReturnModal(true)}
          >
            Initiate Return
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

      {canCreateShipment && (
        <Card>
          <h2 className="text-lg font-semibold mb-3 flex items-center gap-2">
            <Truck className="w-5 h-5" />
            Create Shipment
          </h2>
          <div className="flex flex-col sm:flex-row sm:items-center gap-3">
            <div className="flex items-center gap-2 text-sm text-gray-600">
              Priority:
              <PriorityToggle
                value={priority}
                onChange={setPriority}
                disabled={createShipmentMutation.isPending}
              />
            </div>
            <Button
              onClick={() => createShipmentMutation.mutate({ id: order.id, p: priority })}
              loading={createShipmentMutation.isPending}
              leftIcon={<Truck className="w-4 h-4" />}
            >
              Create shipment
            </Button>
            <p className="text-xs text-gray-500">
              Priority creates an immediate per-order manifest. Normal joins the next 17:00 IST
              pickup batch.
            </p>
          </div>
        </Card>
      )}

      {order.shipment && (
        <Card>
          <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <Truck className="w-5 h-5" />
            Shipping
          </h2>
          <dl className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 text-sm">
            <div>
              <dt className="text-gray-500">Courier</dt>
              <dd className="font-medium">
                {order.shipment.courier_name || order.shipment.provider}
              </dd>
            </div>
            <div>
              <dt className="text-gray-500">AWB</dt>
              <dd className="font-mono">{order.shipment.awb_number || '—'}</dd>
            </div>
            <div>
              <dt className="text-gray-500">Status</dt>
              <dd>
                <Badge variant={getStatusBadgeVariant(order.shipment.status)}>
                  {order.shipment.status}
                </Badge>
              </dd>
            </div>
            <div>
              <dt className="text-gray-500">Priority</dt>
              <dd>
                <PriorityBadge priority={order.shipment.priority} />
              </dd>
            </div>
            <div>
              <dt className="text-gray-500">Manifest ID</dt>
              <dd className="font-mono break-all">{order.shipment.manifest_id || '—'}</dd>
            </div>
            <div>
              <dt className="text-gray-500">Shipping charge</dt>
              <dd>{formatCurrency(order.shipment.shipping_charge)}</dd>
            </div>
            {order.shipment.is_cod && (
              <>
                <div>
                  <dt className="text-gray-500">COD amount</dt>
                  <dd>{formatCurrency(order.shipment.cod_amount)}</dd>
                </div>
                <div>
                  <dt className="text-gray-500">COD remitted</dt>
                  <dd>
                    <Badge variant={order.shipment.cod_remitted ? 'success' : 'warning'}>
                      {order.shipment.cod_remitted ? 'Yes' : 'Pending'}
                    </Badge>
                  </dd>
                </div>
              </>
            )}
            <div>
              <dt className="text-gray-500">NDR attempts</dt>
              <dd>
                <Badge variant={order.shipment.ndr_count > 0 ? 'warning' : 'gray'}>
                  {order.shipment.ndr_count}
                </Badge>
              </dd>
            </div>
            {order.shipment.label_url && (
              <div className="md:col-span-2 lg:col-span-3">
                <a
                  href={order.shipment.label_url}
                  target="_blank"
                  rel="noreferrer"
                  className="text-sm text-primary-600 hover:underline"
                >
                  Download shipping label
                </a>
              </div>
            )}
          </dl>
        </Card>
      )}

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
                      <p className="text-sm text-gray-500">Qty: {item.quantity}</p>
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
              {order.tax_amount > 0 && (
                <div className="flex justify-between">
                  <span className="text-gray-600">Tax</span>
                  <span>{formatCurrency(order.tax_amount)}</span>
                </div>
              )}
              <div className="border-t pt-2 flex justify-between font-semibold text-lg">
                <span>Total</span>
                <span>{formatCurrency(order.total_amount)}</span>
              </div>
            </div>
          </Card>

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
              <Button variant="secondary" size="sm" onClick={() => navigate('/customers')}>
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
              <p className="font-medium">{order.shipping_address?.name}</p>
              <p className="text-gray-600">{order.shipping_address?.street}</p>
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
              </div>
            </Card>
          )}

          {/* Timeline */}
          <OrderTimeline createdAt={order.created_at} updatedAt={order.updated_at} />
        </div>
      </div>

      {/* Update Status Modal */}
      <Modal
        isOpen={showStatusModal}
        onClose={() => setShowStatusModal(false)}
        title="Update Order Status"
        size="sm"
      >
        <div className="space-y-4">
          <Select
            label="New Status"
            options={ORDER_STATUSES.map((s) => ({ value: s, label: s }))}
            value={newStatus}
            onChange={(e) => setNewStatus(e.target.value as OrderStatus)}
          />
          <div className="flex justify-end gap-3 pt-4">
            <Button variant="secondary" onClick={() => setShowStatusModal(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => updateStatusMutation.mutate({ id: order.id, status: newStatus })}
              loading={updateStatusMutation.isPending}
              disabled={!newStatus || newStatus === order.status || updateStatusMutation.isPending}
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
                })
              }
              loading={updateTrackingMutation.isPending}
              disabled={!trackingNumber}
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

      {/* Cancel Order Modal */}
      <ConfirmModal
        isOpen={showCancelModal}
        onClose={() => setShowCancelModal(false)}
        onConfirm={() => cancelMutation.mutate(order.id)}
        title="Cancel Order"
        message={`Are you sure you want to cancel order #${order.order_number || order.id.slice(0, 8)}? This action cannot be undone.`}
        confirmText="Cancel Order"
        loading={cancelMutation.isPending}
      />

      {/* Return Form Modal */}
      <ReturnFormModal
        isOpen={showReturnModal}
        onClose={() => setShowReturnModal(false)}
        order={order}
      />
    </div>
  );
}
