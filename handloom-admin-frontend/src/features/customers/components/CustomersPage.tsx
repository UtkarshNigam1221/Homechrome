import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Edit, Mail, MapPin, Phone, Plus, Search, ShoppingBag, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { customersApi } from '@/features/customers/api';
import {
  Badge,
  Button,
  Card,
  Input,
  Modal,
  PageHeader,
  Pagination,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '@/shared/components/ui';
import { useCursorPagination, useDebounce, useDeleteWithConfirm } from '@/shared/hooks';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency } from '@/shared/utils/currency';

import { addressFullName, customerDisplayName, customerInitial } from '../lib/displayName';
import type { Customer } from '../types';
import { CustomerFormModal } from './CustomerFormModal';

export function CustomersPage() {
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
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null);
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingCustomer, setEditingCustomer] = useState<Customer | null>(null);

  // Fetch customers
  const { data: customersData, isLoading } = useQuery({
    queryKey: ['customers', { limit, cursor, search: debouncedSearch }],
    queryFn: () =>
      customersApi.list({
        limit,
        cursor,
        search: debouncedSearch || undefined,
      }),
  });

  const { setDeleteTarget: setDeleteCustomer, DeleteConfirmModal } = useDeleteWithConfirm<Customer>(
    {
      queryKey: 'customers',
      deleteFn: customersApi.delete,
      entityName: 'Customer',
      getEntityName: (c) => customerDisplayName(c) || c.email,
    }
  );

  const customers = customersData?.items || [];
  const pagination = customersData?.pagination;

  // Deep link from elsewhere in the admin (e.g. an order's "View Customer"):
  // /customers?id=<uuid> opens that customer's detail modal. Fetched directly
  // rather than searched for in the list, which is paginated.
  const [searchParams, setSearchParams] = useSearchParams();
  const deepLinkedId = searchParams.get('id');

  const { data: deepLinkedCustomer } = useQuery({
    queryKey: ['customer', deepLinkedId],
    queryFn: () => customersApi.get(deepLinkedId ?? ''),
    enabled: !!deepLinkedId,
  });

  // Derived rather than synced into state: a row click wins, otherwise show
  // whoever the URL points at.
  const detailCustomer = selectedCustomer ?? deepLinkedCustomer ?? null;

  const closeDetail = () => {
    setSelectedCustomer(null);
    if (deepLinkedId) setSearchParams({}, { replace: true });
  };

  const handleOpenCreate = () => {
    setEditingCustomer(null);
    setShowFormModal(true);
  };

  const handleEdit = (customer: Customer) => {
    setEditingCustomer(customer);
    setShowFormModal(true);
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <PageHeader
        title="Customers"
        subtitle="Manage your customer database"
        action={
          <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
            Add Customer
          </Button>
        }
      />

      {/* Search */}
      <Card padding="sm">
        <Input
          placeholder="Search customers by name, email, phone..."
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            resetPagination();
          }}
          leftIcon={<Search className="w-4 h-4" />}
        />
      </Card>

      {/* Customers Table */}
      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Customer</TableHead>
              <TableHead>Contact</TableHead>
              <TableHead>Orders</TableHead>
              <TableHead>Total Spent</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Joined</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={7} />
            ) : customers.length === 0 ? (
              <TableEmpty
                colSpan={7}
                message="No customers found"
                description={
                  searchQuery
                    ? 'Try a different search term'
                    : 'Customers will appear here when they create accounts'
                }
                action={
                  !searchQuery && (
                    <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
                      Add your first customer
                    </Button>
                  )
                }
              />
            ) : (
              customers.map((customer) => (
                <TableRow key={customer.id} clickable onClick={() => setSelectedCustomer(customer)}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-primary-100 rounded-full flex items-center justify-center">
                        <span className="text-primary-600 font-medium">
                          {customerInitial(customer)}
                        </span>
                      </div>
                      <div>
                        <p className="font-medium text-gray-900">
                          {customerDisplayName(customer) || '—'}
                        </p>
                        <p className="text-sm text-gray-500">{customer.email}</p>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      {customer.phone && (
                        <div className="flex items-center gap-1 text-sm text-gray-500">
                          <Phone className="w-3 h-3" />
                          {customer.phone}
                        </div>
                      )}
                      {customer.addresses?.[0] && (
                        <div className="flex items-center gap-1 text-sm text-gray-500">
                          <MapPin className="w-3 h-3" />
                          {customer.addresses[0].city}
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <ShoppingBag className="w-4 h-4 text-gray-400" />
                      <span>{customer.order_count || 0}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="font-medium">{formatCurrency(customer.total_spent || 0)}</span>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadgeVariant(customer.status)}>
                      {customer.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-gray-500">
                      {format(new Date(customer.created_at), 'MMM d, yyyy')}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEdit(customer);
                        }}
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteCustomer(customer);
                        }}
                        className="text-red-600 hover:text-red-700 hover:bg-red-50"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>

        <div className="border-t border-gray-200 px-6">
          <Pagination
            hasMore={pagination?.has_more ?? false}
            hasPrevious={hasPrevious}
            perPage={limit}
            onNextPage={() => pagination?.next_cursor && goToNextPage(pagination.next_cursor)}
            onPreviousPage={goToPreviousPage}
            onPerPageChange={changeLimit}
            itemCount={customers.length}
          />
        </div>
      </Card>

      {/* Customer Detail Modal */}
      <Modal
        isOpen={!!detailCustomer && !showFormModal}
        onClose={closeDetail}
        title="Customer Details"
        size="md"
      >
        {detailCustomer && (
          <div className="space-y-4">
            <div className="flex items-center gap-4">
              <div className="w-16 h-16 bg-primary-100 rounded-full flex items-center justify-center">
                <span className="text-primary-600 text-xl font-medium">
                  {customerInitial(detailCustomer)}
                </span>
              </div>
              <div>
                <h3 className="text-lg font-semibold">
                  {customerDisplayName(detailCustomer) || '—'}
                </h3>
                <div className="flex items-center gap-2 text-gray-500">
                  <Mail className="w-4 h-4" />
                  {detailCustomer.email}
                </div>
                {detailCustomer.phone && (
                  <div className="flex items-center gap-2 text-gray-500">
                    <Phone className="w-4 h-4" />
                    {detailCustomer.phone}
                  </div>
                )}
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4 pt-4 border-t">
              <div>
                <p className="text-sm text-gray-500">Total Orders</p>
                <p className="text-lg font-semibold">{detailCustomer.order_count || 0}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Total Spent</p>
                <p className="text-lg font-semibold">
                  {formatCurrency(detailCustomer.total_spent || 0)}
                </p>
              </div>
            </div>
            {detailCustomer.addresses && detailCustomer.addresses.length > 0 && (
              <div className="pt-4 border-t">
                <p className="text-sm font-medium text-gray-700 mb-2">Addresses</p>
                {detailCustomer.addresses.map((address, idx) => (
                  <div key={idx} className="text-sm text-gray-600 p-3 bg-gray-50 rounded-lg">
                    <p>{addressFullName(address)}</p>
                    <p>{address.address_line1}</p>
                    {address.address_line2 && <p>{address.address_line2}</p>}
                    <p>
                      {address.city}, {address.state} {address.postal_code}
                    </p>
                    <p>{address.country}</p>
                  </div>
                ))}
              </div>
            )}
            <div className="flex justify-end gap-3 pt-4 border-t">
              <Button
                variant="secondary"
                onClick={() => {
                  setSelectedCustomer(null);
                  handleEdit(detailCustomer);
                }}
              >
                Edit Customer
              </Button>
            </div>
          </div>
        )}
      </Modal>

      {/* Customer Form Modal */}
      <CustomerFormModal
        isOpen={showFormModal}
        onClose={() => {
          setShowFormModal(false);
          setEditingCustomer(null);
        }}
        customer={editingCustomer}
      />

      {/* Delete Confirmation */}
      <DeleteConfirmModal />
    </div>
  );
}
