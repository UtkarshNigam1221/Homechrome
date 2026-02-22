import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Calendar, Download, Eye, Filter, Package, Search } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';
import { useNavigate } from 'react-router-dom';

import { ordersApi } from '@/features/orders/api';
import { getErrorMessage } from '@/shared/api/client';
import {
  Badge,
  Button,
  Card,
  Input,
  Modal,
  Pagination,
  Select,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '@/shared/components/ui';
import { useCursorPagination, useDebounce } from '@/shared/hooks';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency } from '@/shared/utils/currency';

import type { Order, OrderStatus } from '../types';

const ORDER_STATUSES: OrderStatus[] = [
  'PENDING',
  'CONFIRMED',
  'PROCESSING',
  'SHIPPED',
  'DELIVERED',
  'CANCELLED',
  'RETURNED',
];

export function OrdersPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const {
    limit,
    cursor,
    hasPrevious,
    goToNextPage,
    goToPreviousPage,
    resetPagination,
    changeLimit,
  } = useCursorPagination(10);
  const [searchQuery, setSearchQuery] = useState('');
  const debouncedSearch = useDebounce(searchQuery, 300);
  const [statusFilter, setStatusFilter] = useState('');
  const [paymentFilter, setPaymentFilter] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null);
  const [newStatus, setNewStatus] = useState<OrderStatus | ''>('');

  // Fetch orders
  const { data: ordersData, isLoading } = useQuery({
    queryKey: [
      'orders',
      {
        limit,
        cursor,
        search: debouncedSearch,
        status: statusFilter,
        payment_status: paymentFilter,
      },
    ],
    queryFn: () =>
      ordersApi.list({
        limit,
        cursor,
        search: debouncedSearch || undefined,
        status: statusFilter || undefined,
        payment_status: paymentFilter || undefined,
      }),
  });

  // Update status mutation
  const updateStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      ordersApi.updateStatus(id, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
      toast.success('Order status updated');
      setSelectedOrder(null);
      setNewStatus('');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const orders = ordersData?.items || [];
  const pagination = ordersData?.pagination;

  const handleUpdateStatus = () => {
    if (selectedOrder && newStatus) {
      updateStatusMutation.mutate({ id: selectedOrder.id, status: newStatus });
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Orders</h1>
          <p className="page-subtitle">Manage customer orders and fulfillment</p>
        </div>
        <Button variant="secondary" leftIcon={<Download className="w-4 h-4" />}>
          Export Orders
        </Button>
      </div>

      {/* Filters */}
      <Card padding="sm">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1">
            <Input
              placeholder="Search by order number, customer..."
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                resetPagination();
              }}
              leftIcon={<Search className="w-4 h-4" />}
            />
          </div>
          <Button
            variant="secondary"
            leftIcon={<Filter className="w-4 h-4" />}
            onClick={() => setShowFilters(!showFilters)}
          >
            Filters
          </Button>
        </div>

        {showFilters && (
          <div className="mt-4 pt-4 border-t border-gray-200 grid grid-cols-1 md:grid-cols-3 gap-4">
            <Select
              label="Order Status"
              options={[
                { value: '', label: 'All Statuses' },
                ...ORDER_STATUSES.map((s) => ({ value: s, label: s })),
              ]}
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                resetPagination();
              }}
            />
            <Select
              label="Payment Status"
              options={[
                { value: '', label: 'All Payment Statuses' },
                { value: 'PENDING', label: 'Pending' },
                { value: 'PAID', label: 'Paid' },
                { value: 'FAILED', label: 'Failed' },
                { value: 'REFUNDED', label: 'Refunded' },
              ]}
              value={paymentFilter}
              onChange={(e) => {
                setPaymentFilter(e.target.value);
                resetPagination();
              }}
            />
            <div className="flex items-end">
              <Button
                variant="ghost"
                onClick={() => {
                  setStatusFilter('');
                  setPaymentFilter('');
                  setSearchQuery('');
                  resetPagination();
                }}
              >
                Clear Filters
              </Button>
            </div>
          </div>
        )}
      </Card>

      {/* Orders Table */}
      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Order</TableHead>
              <TableHead>Customer</TableHead>
              <TableHead>Items</TableHead>
              <TableHead>Total</TableHead>
              <TableHead>Payment</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Date</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={8} />
            ) : orders.length === 0 ? (
              <TableEmpty
                colSpan={8}
                message="No orders found"
                description={
                  searchQuery || statusFilter || paymentFilter
                    ? 'Try adjusting your filters'
                    : 'Orders will appear here when customers place them'
                }
              />
            ) : (
              orders.map((order) => (
                <TableRow key={order.id} clickable onClick={() => navigate(`/orders/${order.id}`)}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-primary-50 rounded-lg flex items-center justify-center">
                        <Package className="w-5 h-5 text-primary-600" />
                      </div>
                      <div>
                        <p className="font-medium text-gray-900">
                          #{order.order_number || order.id.slice(0, 8)}
                        </p>
                        {order.tracking_number && (
                          <p className="text-sm text-gray-500">{order.tracking_number}</p>
                        )}
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div>
                      <p className="font-medium">{order.customer_name || 'N/A'}</p>
                      <p className="text-sm text-gray-500">{order.customer_email}</p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-gray-600">{order.items?.length || 0} items</span>
                  </TableCell>
                  <TableCell>
                    <span className="font-medium">{formatCurrency(order.total_amount)}</span>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadgeVariant(order.payment_status)}>
                      {order.payment_status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadgeVariant(order.status)}>{order.status}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1 text-sm text-gray-500">
                      <Calendar className="w-4 h-4" />
                      {format(new Date(order.created_at), 'MMM d, yyyy')}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          navigate(`/orders/${order.id}`);
                        }}
                      >
                        <Eye className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setSelectedOrder(order);
                          setNewStatus(order.status);
                        }}
                      >
                        Update
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>

        {(pagination?.has_more || hasPrevious) && (
          <div className="border-t border-gray-200 px-6">
            <Pagination
              hasMore={pagination?.has_more ?? false}
              hasPrevious={hasPrevious}
              perPage={limit}
              onNextPage={() => pagination?.next_cursor && goToNextPage(pagination.next_cursor)}
              onPreviousPage={goToPreviousPage}
              onPerPageChange={changeLimit}
              itemCount={orders.length}
            />
          </div>
        )}
      </Card>

      {/* Update Status Modal */}
      <Modal
        isOpen={!!selectedOrder}
        onClose={() => {
          setSelectedOrder(null);
          setNewStatus('');
        }}
        title="Update Order Status"
        size="sm"
      >
        <div className="space-y-4">
          <div>
            <p className="text-sm text-gray-600 mb-2">
              Order:{' '}
              <span className="font-medium">
                #{selectedOrder?.order_number || selectedOrder?.id.slice(0, 8)}
              </span>
            </p>
            <p className="text-sm text-gray-600">
              Current Status:{' '}
              <Badge variant={getStatusBadgeVariant(selectedOrder?.status || '')}>
                {selectedOrder?.status}
              </Badge>
            </p>
          </div>
          <Select
            label="New Status"
            options={ORDER_STATUSES.map((s) => ({ value: s, label: s }))}
            value={newStatus}
            onChange={(e) => setNewStatus(e.target.value as OrderStatus)}
          />
          <div className="flex justify-end gap-3 pt-4">
            <Button variant="secondary" onClick={() => setSelectedOrder(null)}>
              Cancel
            </Button>
            <Button
              onClick={handleUpdateStatus}
              loading={updateStatusMutation.isPending}
              disabled={!newStatus || newStatus === selectedOrder?.status}
            >
              Update Status
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
