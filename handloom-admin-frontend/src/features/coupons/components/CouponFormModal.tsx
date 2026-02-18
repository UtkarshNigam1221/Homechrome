import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { couponsApi } from '@/features/coupons/api';
import { getErrorMessage } from '@/shared/api/client';
import { Button, Input, Modal, Select } from '@/shared/components/ui';

import type { Coupon, CreateCouponRequest } from '../types';

const couponSchema = z.object({
  code: z
    .string()
    .min(3, 'Code must be at least 3 characters')
    .max(20, 'Code must be less than 20 characters')
    .regex(
      /^[A-Z0-9_-]+$/,
      'Code must be uppercase letters, numbers, underscores, and hyphens only'
    ),
  type: z.enum(['PERCENTAGE', 'FIXED_AMOUNT', 'FREE_SHIPPING']),
  discount_value: z.number().min(0, 'Discount value must be positive'),
  max_uses: z.number().min(0).optional(),
  min_order_value: z.number().min(0).optional(),
  max_discount: z.number().min(0).optional(),
  expiry_date: z.string().optional(),
  status: z.enum(['ACTIVE', 'INACTIVE', 'EXPIRED']),
});

type CouponFormData = z.infer<typeof couponSchema>;

interface CouponFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  coupon?: Coupon | null;
}

export function CouponFormModal({ isOpen, onClose, coupon }: CouponFormModalProps) {
  const queryClient = useQueryClient();
  const isEditing = !!coupon?.id;

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<CouponFormData>({
    resolver: zodResolver(couponSchema),
    defaultValues: {
      code: '',
      type: 'PERCENTAGE',
      discount_value: 10,
      max_uses: 0,
      min_order_value: 0,
      max_discount: 0,
      expiry_date: '',
      status: 'ACTIVE',
    },
  });

  const couponType = watch('type');

  // Reset form when modal opens/closes or coupon changes
  useEffect(() => {
    if (isOpen) {
      if (coupon?.id) {
        reset({
          code: coupon.code,
          type: coupon.type,
          discount_value:
            coupon.type === 'FIXED_AMOUNT' ? coupon.discount_value / 100 : coupon.discount_value,
          max_uses: coupon.max_uses || 0,
          min_order_value: coupon.min_order_value ? coupon.min_order_value / 100 : 0,
          max_discount: coupon.max_discount ? coupon.max_discount / 100 : 0,
          expiry_date: coupon.expiry_date ? coupon.expiry_date.split('T')[0] : '',
          status: coupon.status,
        });
      } else {
        reset({
          code: '',
          type: 'PERCENTAGE',
          discount_value: 10,
          max_uses: 0,
          min_order_value: 0,
          max_discount: 0,
          expiry_date: '',
          status: 'ACTIVE',
        });
      }
    }
  }, [isOpen, coupon, reset]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateCouponRequest) => couponsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coupons'] });
      toast.success('Coupon created successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateCouponRequest> }) =>
      couponsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['coupons'] });
      toast.success('Coupon updated successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: CouponFormData) => {
    const requestData: CreateCouponRequest = {
      code: data.code.toUpperCase(),
      type: data.type,
      discount_value:
        data.type === 'FIXED_AMOUNT' ? Math.round(data.discount_value * 100) : data.discount_value,
      max_uses: data.max_uses || undefined,
      min_order_value: data.min_order_value ? Math.round(data.min_order_value * 100) : undefined,
      max_discount: data.max_discount ? Math.round(data.max_discount * 100) : undefined,
      expiry_date: data.expiry_date || undefined,
      status: data.status,
    };

    if (isEditing && coupon?.id) {
      updateMutation.mutate({ id: coupon.id, data: requestData });
    } else {
      createMutation.mutate(requestData);
    }
  };

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Coupon' : 'Create Coupon'}
      size="md"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <Input
          label="Coupon Code"
          placeholder="e.g., SUMMER20"
          hint="Uppercase letters, numbers, underscores, and hyphens only"
          error={errors.code?.message}
          required
          {...register('code')}
        />

        <Select
          label="Discount Type"
          options={[
            { value: 'PERCENTAGE', label: 'Percentage Discount' },
            { value: 'FIXED_AMOUNT', label: 'Fixed Amount Off' },
            { value: 'FREE_SHIPPING', label: 'Free Shipping' },
          ]}
          error={errors.type?.message}
          required
          {...register('type')}
        />

        {couponType !== 'FREE_SHIPPING' && (
          <Input
            label={couponType === 'PERCENTAGE' ? 'Discount Percentage' : 'Discount Amount (₹)'}
            type="number"
            step={couponType === 'PERCENTAGE' ? '1' : '0.01'}
            min="0"
            max={couponType === 'PERCENTAGE' ? '100' : undefined}
            placeholder={couponType === 'PERCENTAGE' ? '10' : '100'}
            error={errors.discount_value?.message}
            required
            {...register('discount_value', { valueAsNumber: true })}
          />
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Maximum Uses"
            type="number"
            min="0"
            placeholder="Unlimited"
            hint="Leave 0 for unlimited"
            error={errors.max_uses?.message}
            {...register('max_uses', { valueAsNumber: true })}
          />

          <Input
            label="Minimum Order Value (₹)"
            type="number"
            step="0.01"
            min="0"
            placeholder="No minimum"
            error={errors.min_order_value?.message}
            {...register('min_order_value', { valueAsNumber: true })}
          />
        </div>

        {couponType === 'PERCENTAGE' && (
          <Input
            label="Maximum Discount (₹)"
            type="number"
            step="0.01"
            min="0"
            placeholder="No cap"
            hint="Cap the maximum discount amount"
            error={errors.max_discount?.message}
            {...register('max_discount', { valueAsNumber: true })}
          />
        )}

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Expiry Date"
            type="date"
            error={errors.expiry_date?.message}
            {...register('expiry_date')}
          />

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
        </div>

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
