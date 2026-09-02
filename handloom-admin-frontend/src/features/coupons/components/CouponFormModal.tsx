import { zodResolver } from '@hookform/resolvers/zod';
import { format } from 'date-fns';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';

import { couponsApi } from '@/features/coupons/api';
import { Button, Input, Modal, Select, type SelectOption } from '@/shared/components/ui';
import { useFormMutation } from '@/shared/hooks';

import { couponSchema } from '../lib/couponSchema';
import {
  type CouponFormValues,
  couponToFormValues,
  defaultCouponFormValues,
  toCreateRequest,
  toUpdateRequest,
} from '../lib/toCreateRequest';
import type { Coupon, CreateCouponRequest, UpdateCouponRequest } from '../types';

const audienceOptions: SelectOption[] = [
  { value: 'ALL', label: 'Everyone' },
  { value: 'FIRST_ORDER', label: 'First order only' },
  { value: 'RETURNING', label: 'Returning customers' },
  { value: 'SPECIFIC_CUSTOMER', label: 'One specific customer' },
];

interface CouponFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  coupon?: Coupon | null;
}

export function CouponFormModal({ isOpen, onClose, coupon }: CouponFormModalProps) {
  const isEditing = !!coupon?.id;
  // The API can't change these post-creation (see toUpdateRequest) — disabled, not just
  // decorative, so an operator can't "save" an edit the backend silently drops.
  const lockedAfterCreate = isEditing;

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<CouponFormValues>({
    resolver: zodResolver(couponSchema),
    defaultValues: defaultCouponFormValues,
  });

  const type = watch('type');
  const audience = watch('audience');
  const noEndDate = watch('noEndDate');

  // Reset the form when the modal opens, or when it switches between create and edit.
  useEffect(() => {
    if (!isOpen) return;
    if (coupon) {
      reset(couponToFormValues(coupon));
    } else {
      reset({ ...defaultCouponFormValues, validFrom: format(new Date(), 'yyyy-MM-dd') });
    }
  }, [isOpen, coupon, reset]);

  const { isLoading, onSubmit: submitMutation } = useFormMutation<
    CreateCouponRequest,
    UpdateCouponRequest
  >({
    queryKey: 'coupons',
    createFn: couponsApi.create,
    updateFn: couponsApi.update,
    entityName: 'Coupon',
    onSuccess: onClose,
  });

  const onSubmit = (data: CouponFormValues) => {
    submitMutation(coupon?.id, toCreateRequest(data), toUpdateRequest(data));
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Coupon' : 'Create Coupon'}
      size="lg"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Coupon Code"
            placeholder="e.g., SUMMER20"
            hint={
              lockedAfterCreate
                ? "Can't be changed after creation"
                : 'Uppercase letters, numbers, underscores, and hyphens only'
            }
            error={errors.code?.message}
            required
            disabled={lockedAfterCreate}
            {...register('code')}
          />

          <Input
            label="Name"
            placeholder="e.g., Summer Sale 20%"
            error={errors.name?.message}
            required
            {...register('name')}
          />
        </div>

        <div>
          <label className="label">Description</label>
          <textarea
            className="input min-h-[70px]"
            placeholder="Optional — shown to the operator, not the customer"
            {...register('description')}
          />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Select
            label="Discount Type"
            options={[
              { value: 'PERCENTAGE', label: 'Percentage Discount' },
              { value: 'FIXED', label: 'Fixed Amount Off' },
            ]}
            error={errors.type?.message}
            required
            disabled={lockedAfterCreate}
            hint={lockedAfterCreate ? "Can't be changed after creation" : undefined}
            {...register('type')}
          />

          <Input
            label={type === 'PERCENTAGE' ? 'Discount Percentage' : 'Discount Amount (₹)'}
            type="number"
            step={type === 'PERCENTAGE' ? '1' : '0.01'}
            min="0"
            max={type === 'PERCENTAGE' ? '100' : undefined}
            placeholder={type === 'PERCENTAGE' ? '10' : '100'}
            error={errors.value?.message}
            required
            disabled={lockedAfterCreate}
            hint={lockedAfterCreate ? "Can't be changed after creation" : undefined}
            {...register('value', { valueAsNumber: true })}
          />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Minimum Order Value (₹)"
            type="number"
            step="0.01"
            min="0"
            placeholder="No minimum"
            error={errors.minOrderValue?.message}
            {...register('minOrderValue', { valueAsNumber: true })}
          />

          {type === 'PERCENTAGE' && (
            <Input
              label="Maximum Discount (₹)"
              type="number"
              step="0.01"
              min="0"
              placeholder="No cap"
              hint="Caps the discount amount — meaningless on a fixed-amount coupon"
              error={errors.maxDiscount?.message}
              {...register('maxDiscount', { valueAsNumber: true })}
            />
          )}
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Usage Limit"
            type="number"
            min="0"
            placeholder="Unlimited"
            hint="Leave 0 for unlimited"
            error={errors.usageLimit?.message}
            {...register('usageLimit', { valueAsNumber: true })}
          />

          <Input
            label="Uses Per Customer"
            type="number"
            min="0"
            placeholder="Unlimited"
            hint="Leave 0 for unlimited"
            error={errors.usagePerUser?.message}
            {...register('usagePerUser', { valueAsNumber: true })}
          />
        </div>

        <Select
          label="Who can use this coupon"
          options={audienceOptions}
          error={errors.audience?.message}
          required
          disabled={lockedAfterCreate}
          hint={lockedAfterCreate ? "Can't be changed after creation" : undefined}
          {...register('audience')}
        />

        {audience === 'SPECIFIC_CUSTOMER' && !isEditing && (
          <Input
            label="Customer mobile number"
            placeholder="98765 43210"
            hint="The number the customer signs in with."
            leftIcon={<span className="text-sm font-medium text-gray-600">+91</span>}
            inputMode="numeric"
            error={errors.customerPhone?.message}
            {...register('customerPhone')}
          />
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Valid From"
            type="date"
            error={errors.validFrom?.message}
            required
            {...register('validFrom')}
          />

          <Input
            label="Expiry Date"
            type="date"
            error={errors.expiryDate?.message}
            disabled={noEndDate}
            {...register('expiryDate')}
          />
        </div>

        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="noEndDate"
            className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            {...register('noEndDate')}
          />
          <label htmlFor="noEndDate" className="text-sm text-gray-700">
            No end date — runs until switched off
          </label>
        </div>

        <div>
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="combinesWithOffers"
              className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              {...register('combinesWithOffers')}
            />
            <label htmlFor="combinesWithOffers" className="text-sm text-gray-700">
              Combines with automatic offers
            </label>
          </div>
          <p className="mt-1 ml-7 text-xs text-gray-500">
            Off by default. A buy-2-get-1 offer is already a third off, so a 20% code on top is
            46.7% off.
          </p>
        </div>

        {isEditing && (
          <Select
            label="Status"
            options={[
              { value: 'ACTIVE', label: 'Active' },
              { value: 'INACTIVE', label: 'Inactive' },
              { value: 'EXPIRED', label: 'Expired' },
            ]}
            required
            {...register('status')}
          />
        )}

        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Update Coupon' : 'Create Coupon'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
