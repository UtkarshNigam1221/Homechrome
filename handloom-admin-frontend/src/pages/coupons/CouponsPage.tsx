import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Calendar, Edit, Percent, Plus, Search, Trash2 } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { couponsApi, getErrorMessage } from '../../api';
import {
  Badge,
  Button,
  Card,
  ConfirmModal,
  getStatusBadgeVariant,
  Input,
  Pagination,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '../../components/common';
import { useCursorPagination } from '../../hooks';
import type { Coupon } from '../../types';
import { CouponFormModal } from './CouponFormModal';

export function CouponsPage() {
  const queryClient = useQueryClient();
  const { limit, cursor, hasPrevious, goToNextPage, goToPreviousPage, resetPagination, changeLimit } = useCursorPagination(10);
  const [searchQuery, setSearchQuery] = useState('');
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingCoupon, setEditingCoupon] = useState<Coupon | null>(null);
  const [deleteCoupon, setDeleteCoupon] = useState<Coupon | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['coupons', { limit, cursor, search: searchQuery }],
    queryFn: () => couponsApi.list({ limit, cursor, search: searchQuery || undefined }),
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: couponsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coupons'] });
      toast.success('Coupon deleted successfully');
      setDeleteCoupon(null);
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const coupons = data?.items || [];
  const pagination = data?.pagination;

  const handleOpenCreate = () => {
    setEditingCoupon(null);
    setShowFormModal(true);
  };

  const handleEdit = (coupon: Coupon) => {
    setEditingCoupon(coupon);
    setShowFormModal(true);
  };

  const formatDiscount = (coupon: Coupon) => {
    if (coupon.type === 'PERCENTAGE') return `${coupon.discount_value}% off`;
    if (coupon.type === 'FIXED_AMOUNT') return `₹${coupon.discount_value / 100} off`;
    return 'Free Shipping';
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Coupons</h1>
          <p className="page-subtitle">Manage discount codes and promotions</p>
        </div>
        <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
          Create Coupon
        </Button>
      </div>

      <Card padding="sm">
        <Input
          placeholder="Search coupons by code..."
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            resetPagination();
          }}
          leftIcon={<Search className="w-4 h-4" />}
        />
      </Card>

      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Code</TableHead>
              <TableHead>Discount</TableHead>
              <TableHead>Usage</TableHead>
              <TableHead>Min Order</TableHead>
              <TableHead>Expiry</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={7} />
            ) : coupons.length === 0 ? (
              <TableEmpty
                colSpan={7}
                message="No coupons found"
                action={
                  <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
                    Create your first coupon
                  </Button>
                }
              />
            ) : (
              coupons.map((coupon) => (
                <TableRow key={coupon.id} clickable onClick={() => handleEdit(coupon)}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-purple-50 rounded-lg flex items-center justify-center">
                        <Percent className="w-5 h-5 text-purple-600" />
                      </div>
                      <div>
                        <p className="font-mono font-medium">{coupon.code}</p>
                        <Badge variant="gray" size="sm">
                          {coupon.type.replace('_', ' ')}
                        </Badge>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="font-medium text-green-600">{formatDiscount(coupon)}</span>
                  </TableCell>
                  <TableCell>
                    {coupon.used_count || 0} / {coupon.max_uses || '∞'}
                  </TableCell>
                  <TableCell>
                    {coupon.min_order_value ? `₹${coupon.min_order_value / 100}` : '-'}
                  </TableCell>
                  <TableCell>
                    {coupon.expiry_date ? (
                      <div className="flex items-center gap-1 text-sm">
                        <Calendar className="w-3 h-3" />
                        {format(new Date(coupon.expiry_date), 'MMM d, yyyy')}
                      </div>
                    ) : (
                      <span className="text-gray-400">No expiry</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadgeVariant(coupon.status)}>{coupon.status}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEdit(coupon);
                        }}
                        title="Edit coupon"
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          setDeleteCoupon(coupon);
                        }}
                        title="Delete coupon"
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
              itemCount={coupons.length}
            />
          </div>
        )}
      </Card>

      {/* Coupon Form Modal */}
      <CouponFormModal
        isOpen={showFormModal}
        onClose={() => {
          setShowFormModal(false);
          setEditingCoupon(null);
        }}
        coupon={editingCoupon}
      />

      {/* Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={!!deleteCoupon}
        onClose={() => setDeleteCoupon(null)}
        onConfirm={() => deleteCoupon && deleteMutation.mutate(deleteCoupon.id)}
        title="Delete Coupon"
        message={`Are you sure you want to delete coupon "${deleteCoupon?.code}"? This action cannot be undone.`}
        confirmText="Delete"
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
