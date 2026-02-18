import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Edit, Mail, MapPin, Phone, Plus, Search, ShoppingBag, Trash2 } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { customersApi, getErrorMessage } from '@/api';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
  getStatusBadgeVariant,
  Input,
  Modal,
  Pagination,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '@/components/common';
import { useCursorPagination, useDebounce } from '@/hooks';
import type { Customer } from '@/types';
import { formatCurrency } from '@/utils/currency';

import { CustomerFormModal } from './CustomerFormModal';

export function CustomersPage() {
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
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null);
  const [deleteCustomer, setDeleteCustomer] = useState<Customer | null>(null);
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

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: customersApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] });
      toast.success('Customer deleted successfully');
      setDeleteCustomer(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const customers = customersData?.items || [];
  const pagination = customersData?.pagination;

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
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Customers</h1>
          <p className="page-subtitle">Manage your customer database</p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
          Add Customer
        </Button>
      </div>

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
                          {customer.name?.charAt(0).toUpperCase() || 'C'}
                        </span>
                      </div>
                      <div>
                        <p className="font-medium text-gray-900">{customer.name}</p>
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

        {(pagination?.has_more || hasPrevious) && (
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
        )}
      </Card>

      {/* Customer Detail Modal */}
      <Modal
        isOpen={!!selectedCustomer && !showFormModal}
        onClose={() => setSelectedCustomer(null)}
        title="Customer Details"
        size="md"
      >
        {selectedCustomer && (
          <div className="space-y-4">
            <div className="flex items-center gap-4">
              <div className="w-16 h-16 bg-primary-100 rounded-full flex items-center justify-center">
                <span className="text-primary-600 text-xl font-medium">
                  {selectedCustomer.name?.charAt(0).toUpperCase() || 'C'}
                </span>
              </div>
              <div>
                <h3 className="text-lg font-semibold">{selectedCustomer.name}</h3>
                <div className="flex items-center gap-2 text-gray-500">
                  <Mail className="w-4 h-4" />
                  {selectedCustomer.email}
                </div>
                {selectedCustomer.phone && (
                  <div className="flex items-center gap-2 text-gray-500">
                    <Phone className="w-4 h-4" />
                    {selectedCustomer.phone}
                  </div>
                )}
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4 pt-4 border-t">
              <div>
                <p className="text-sm text-gray-500">Total Orders</p>
                <p className="text-lg font-semibold">{selectedCustomer.order_count || 0}</p>
              </div>
              <div>
                <p className="text-sm text-gray-500">Total Spent</p>
                <p className="text-lg font-semibold">
                  {formatCurrency(selectedCustomer.total_spent || 0)}
                </p>
              </div>
            </div>
            {selectedCustomer.addresses && selectedCustomer.addresses.length > 0 && (
              <div className="pt-4 border-t">
                <p className="text-sm font-medium text-gray-700 mb-2">Addresses</p>
                {selectedCustomer.addresses.map((address, idx) => (
                  <div key={idx} className="text-sm text-gray-600 p-3 bg-gray-50 rounded-lg">
                    <p>{address.name}</p>
                    <p>{address.street}</p>
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
                  handleEdit(selectedCustomer);
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
      <ConfirmModal
        isOpen={!!deleteCustomer}
        onClose={() => setDeleteCustomer(null)}
        onConfirm={() => deleteCustomer && deleteMutation.mutate(deleteCustomer.id)}
        title="Delete Customer"
        message={`Are you sure you want to delete "${deleteCustomer?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
