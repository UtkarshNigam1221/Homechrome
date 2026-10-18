import { useMutation, useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Calendar, Edit, Percent, Plus, Search, Trash2 } from 'lucide-react';
import { useState } from 'react';

import { couponsApi } from '@/features/coupons/api';
import { getErrorMessage } from '@/shared/api/client';
import {
  Badge,
  Button,
  Card,
  Input,
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

import { fromStoredAmount } from '../lib/toCreateRequest';
import type { Coupon } from '../types';
import { CouponFormModal } from './CouponFormModal';

// Formats through noon, not the stored midnight-UTC instant directly, so the calendar
// date shown can't shift a day in a timezone west of UTC.
function formatValidUntil(instant: string): string {
  return format(new Date(`${instant.slice(0, 10)}T12:00:00`), 'MMM d, yyyy');
}

export function CouponsPage() {
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
  const [showFormModal, setShowFormModal] = useState(false);
  const [editingCoupon, setEditingCoupon] = useState<Coupon | null>(null);
  const [findCode, setFindCode] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['coupons', { limit, cursor, search: debouncedSearch }],
    queryFn: () => couponsApi.list({ limit, cursor, search: debouncedSearch || undefined }),
  });

  const { setDeleteTarget: setDeleteCoupon, DeleteConfirmModal } = useDeleteWithConfirm<Coupon>({
    queryKey: 'coupons',
    deleteFn: couponsApi.delete,
    entityName: 'Coupon',
    getEntityName: (c) => c.code,
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

  // A SPECIFIC_CUSTOMER coupon lives in a GSI1 partition this page's list never
  // queries, so a code lookup is the only way to reach it once it's created.
  const findByCodeMutation = useMutation({
    mutationFn: (code: string) => couponsApi.getByCode(code),
    onSuccess: (coupon) => {
      setFindCode('');
      handleEdit(coupon);
    },
  });

  const handleFindByCode = () => {
    const code = findCode.trim();
    if (code) findByCodeMutation.mutate(code);
  };

  const formatDiscount = (coupon: Coupon) => {
    if (coupon.type === 'PERCENTAGE') return `${fromStoredAmount(coupon.value)}% off`;
    return `${formatCurrency(coupon.value)} off`;
  };

  // A promo nearing its cap should be visible before it runs out, not discovered when
  // it silently stops applying.
  const usageBadge = (coupon: Coupon) => {
    if (!coupon.usage_limit) return null;
    const ratio = coupon.usage_count / coupon.usage_limit;
    if (ratio >= 1) return <Badge variant="danger">Exhausted</Badge>;
    if (ratio >= 0.8) return <Badge variant="warning">Near limit</Badge>;
    return null;
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Coupons"
        subtitle="Manage discount codes and promotions"
        action={
          <Button leftIcon={<Plus className="w-4 h-4" />} onClick={handleOpenCreate}>
            Create Coupon
          </Button>
        }
      />

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

      <Card padding="sm">
        <div className="flex flex-col sm:flex-row sm:items-end gap-3">
          <div className="flex-1">
            <Input
              label="Find a coupon by code"
              placeholder="e.g., APOLOGY50"
              hint="Targeted coupons don't appear in the list below — look one up here to edit or switch it off."
              value={findCode}
              onChange={(e) => setFindCode(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleFindByCode()}
              error={
                findByCodeMutation.isError ? getErrorMessage(findByCodeMutation.error) : undefined
              }
            />
          </div>
          <Button
            variant="secondary"
            onClick={handleFindByCode}
            loading={findByCodeMutation.isPending}
            disabled={!findCode.trim()}
          >
            Find
          </Button>
        </div>
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
                          {coupon.type}
                        </Badge>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="font-medium text-green-600">{formatDiscount(coupon)}</span>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span>
                        {coupon.usage_count || 0} / {coupon.usage_limit || '∞'}
                      </span>
                      {usageBadge(coupon)}
                    </div>
                  </TableCell>
                  <TableCell>
                    {coupon.min_order_value ? formatCurrency(coupon.min_order_value) : '-'}
                  </TableCell>
                  <TableCell>
                    {coupon.valid_until ? (
                      <div className="flex items-center gap-1 text-sm">
                        <Calendar className="w-3 h-3" />
                        {formatValidUntil(coupon.valid_until)}
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
      <DeleteConfirmModal />
    </div>
  );
}
