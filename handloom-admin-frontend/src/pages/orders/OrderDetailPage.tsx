import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import {
  ArrowLeft,
  Ban,
  Clock,
  CreditCard,
  Edit,
  MapPin,
  MessageSquare,
  Package,
  Truck,
} from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { useNavigate, useParams } from 'react-router-dom';

import { getErrorMessage, ordersApi } from '@/api';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
  getStatusBadgeVariant,
  Input,
  Modal,
  Select,
} from '@/components/common';
import type { OrderStatus } from '@/types';
import { formatCurrency } from '@/utils/currency';

const ORDER_STATUSES: OrderStatus[] = [
  'PENDING',
  'CONFIRMED',
  'PROCESSING',
  'SHIPPED',
  'DELIVERED',
  'CANCELLED',
  'RETURNED',
];

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

  const canCancel = ['PENDING', 'CONFIRMED'].includes(order.order_status);

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
          <Badge variant={getStatusBadgeVariant(order.order_status)}>{order.order_status}</Badge>
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
            setNewStatus(order.order_status);
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
            setCarrier(order.carrier || '');
            setShowTrackingModal(true);
          }}
        >
          Update Tracking
        </Button>
        <Button
          variant="secondary"
          leftIcon={<MessageSquare className="w-4 h-4" />}
          onClick={() => setShowNoteModal(true)}
        >
          Add Note
        </Button>
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
                      <p className="text-sm text-gray-500">SKU: {item.sku}</p>
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
              {order.discount > 0 && (
                <div className="flex justify-between text-green-600">
                  <span>Discount {order.coupon_code && `(${order.coupon_code})`}</span>
                  <span>-{formatCurrency(order.discount)}</span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-gray-600">Shipping</span>
                <span>{formatCurrency(order.shipping_cost)}</span>
              </div>
              {order.tax > 0 && (
                <div className="flex justify-between">
                  <span className="text-gray-600">Tax</span>
                  <span>{formatCurrency(order.tax)}</span>
                </div>
              )}
              <div className="border-t pt-2 flex justify-between font-semibold text-lg">
                <span>Total</span>
                <span>{formatCurrency(order.total_price)}</span>
              </div>
            </div>
          </Card>

          {/* Notes */}
          {order.notes && order.notes.length > 0 && (
            <Card>
              <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
                <MessageSquare className="w-5 h-5" />
                Order Notes
              </h2>
              <div className="space-y-3">
                {order.notes.map((note, index) => (
                  <div
                    key={note.id || index}
                    className={`p-3 rounded-lg ${note.is_internal ? 'bg-yellow-50 border border-yellow-200' : 'bg-gray-50'}`}
                  >
                    <p className="text-sm">{note.note}</p>
                    <div className="flex items-center gap-2 mt-2 text-xs text-gray-500">
                      <span>{note.created_by}</span>
                      <span>•</span>
                      <span>{format(new Date(note.created_at), 'MMM d, yyyy h:mm a')}</span>
                      {note.is_internal && (
                        <Badge variant="warning" size="sm">
                          Internal
                        </Badge>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </Card>
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
                onClick={() => navigate('/customers')}
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
                  <p className="font-medium">{order.carrier || 'N/A'}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">Tracking Number</p>
                  <p className="font-mono text-sm">{order.tracking_number}</p>
                </div>
              </div>
            </Card>
          )}

          {/* Timeline */}
          <Card>
            <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
              <Clock className="w-5 h-5" />
              Timeline
            </h2>
            <div className="space-y-3">
              <div className="flex items-start gap-3">
                <div className="w-2 h-2 bg-primary-600 rounded-full mt-2" />
                <div>
                  <p className="text-sm font-medium">Order Placed</p>
                  <p className="text-xs text-gray-500">
                    {format(new Date(order.created_at), 'MMM d, yyyy h:mm a')}
                  </p>
                </div>
              </div>
              {order.updated_at !== order.created_at && (
                <div className="flex items-start gap-3">
                  <div className="w-2 h-2 bg-gray-400 rounded-full mt-2" />
                  <div>
                    <p className="text-sm font-medium">Last Updated</p>
                    <p className="text-xs text-gray-500">
                      {format(new Date(order.updated_at), 'MMM d, yyyy h:mm a')}
                    </p>
                  </div>
                </div>
              )}
            </div>
          </Card>
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
              disabled={!newStatus || newStatus === order.order_status}
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
    </div>
  );
}
