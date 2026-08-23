import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { categoriesApi } from '@/features/categories/api';
import { couponsApi } from '@/features/coupons/api';
import { productsApi } from '@/features/products/api';
import { Button, Input, Modal, MultiSelect, Select } from '@/shared/components/ui';
import { useFormMutation } from '@/shared/hooks';

import { toCreateRequest, toFormValues } from '../lib/toCreateRequest';
import type { Coupon, CreateCouponRequest } from '../types';

const couponSchema = z
  .object({
    code: z
      .string()
      .min(3, 'Code must be at least 3 characters')
      .max(20, 'Code must be less than 20 characters')
      .regex(
        /^[A-Z0-9_-]+$/,
        'Code must be uppercase letters, numbers, underscores, and hyphens only'
      ),
    name: z.string().min(1, 'Name is required'),
    description: z.string().optional(),
    type: z.enum(['PERCENTAGE', 'FIXED']),
    value: z.number().gt(0, 'Discount must be greater than zero'),
    min_order_value: z.number().min(0).optional(),
    max_discount: z.number().min(0).optional(),
    usage_limit: z.number().min(0).optional(),
    usage_per_user: z.number().min(0).optional(),
    applicable_categories: z.array(z.string()).optional(),
    applicable_products: z.array(z.string()).optional(),
    excluded_categories: z.array(z.string()).optional(),
    excluded_products: z.array(z.string()).optional(),
    valid_from: z.string().min(1, 'Start date is required'),
    valid_until: z.string().min(1, 'End date is required'),
    status: z.enum(['ACTIVE', 'INACTIVE', 'EXPIRED']),
  })
  .refine((v) => v.type !== 'PERCENTAGE' || v.value <= 100, {
    message: 'A percentage cannot exceed 100',
    path: ['value'],
  })
  .refine((v) => new Date(v.valid_until) >= new Date(v.valid_from), {
    message: 'End date must be on or after the start date',
    path: ['valid_until'],
  });

type CouponFormData = z.infer<typeof couponSchema>;

const today = () => new Date().toISOString().split('T')[0];

const emptyForm = (): CouponFormData => ({
  code: '',
  name: '',
  description: '',
  type: 'PERCENTAGE',
  value: 10,
  min_order_value: 0,
  max_discount: 0,
  usage_limit: 0,
  usage_per_user: 0,
  applicable_categories: [],
  applicable_products: [],
  excluded_categories: [],
  excluded_products: [],
  valid_from: today(),
  valid_until: today(),
  status: 'ACTIVE',
});

interface CouponFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  coupon?: Coupon | null;
}

export function CouponFormModal({ isOpen, onClose, coupon }: CouponFormModalProps) {
  const isEditing = !!coupon?.id;

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<CouponFormData>({
    resolver: zodResolver(couponSchema),
    defaultValues: emptyForm(),
  });

  const couponType = watch('type');

  // Scoping needs something to scope to. Only fetched while the modal is open.
  const { data: categories } = useQuery({
    queryKey: ['categories', 'coupon-scope'],
    queryFn: () => categoriesApi.list({ limit: 200 }),
    enabled: isOpen,
  });
  const { data: products } = useQuery({
    queryKey: ['products', 'coupon-scope'],
    queryFn: () => productsApi.list({ limit: 200 }),
    enabled: isOpen,
  });

  const categoryOptions = (categories?.items ?? []).map((c) => ({ value: c.id, label: c.name }));
  const productOptions = (products?.items ?? []).map((p) => ({
    value: p.id,
    label: `${p.name} (${p.sku})`,
  }));

  useEffect(() => {
    if (!isOpen) return;
    if (!coupon?.id) {
      reset(emptyForm());
      return;
    }
    reset(toFormValues(coupon));
  }, [isOpen, coupon, reset]);

  const { isLoading, onSubmit: submitMutation } = useFormMutation<
    CreateCouponRequest,
    Partial<CreateCouponRequest>
  >({
    queryKey: 'coupons',
    createFn: couponsApi.create,
    updateFn: couponsApi.update,
    entityName: 'Coupon',
    onSuccess: onClose,
  });

  const onSubmit = (data: CouponFormData) => {
    const requestData = toCreateRequest(data);
    submitMutation(coupon?.id, requestData, requestData);
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
            hint="Uppercase letters, numbers, underscores and hyphens"
            error={errors.code?.message}
            required
            {...register('code')}
          />
          <Input
            label="Name"
            placeholder="e.g., Summer Sale 20% off"
            hint="Shown in the admin, not to customers"
            error={errors.name?.message}
            required
            {...register('name')}
          />
        </div>

        <Input
          label="Description"
          placeholder="Optional note about this campaign"
          error={errors.description?.message}
          {...register('description')}
        />

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Select
            label="Discount Type"
            options={[
              { value: 'PERCENTAGE', label: 'Percentage Discount' },
              { value: 'FIXED', label: 'Fixed Amount Off' },
            ]}
            error={errors.type?.message}
            required
            {...register('type')}
          />
          <Input
            label={couponType === 'PERCENTAGE' ? 'Discount Percentage (%)' : 'Discount Amount (₹)'}
            type="number"
            step="0.01"
            min="0"
            max={couponType === 'PERCENTAGE' ? '100' : undefined}
            placeholder={couponType === 'PERCENTAGE' ? '10' : '100'}
            error={errors.value?.message}
            required
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
            error={errors.min_order_value?.message}
            {...register('min_order_value', { valueAsNumber: true })}
          />
          {couponType === 'PERCENTAGE' && (
            <Input
              label="Maximum Discount (₹)"
              type="number"
              step="0.01"
              min="0"
              placeholder="No cap"
              hint="Caps what a percentage can take off"
              error={errors.max_discount?.message}
              {...register('max_discount', { valueAsNumber: true })}
            />
          )}
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Total Uses"
            type="number"
            min="0"
            placeholder="Unlimited"
            hint="0 for unlimited"
            error={errors.usage_limit?.message}
            {...register('usage_limit', { valueAsNumber: true })}
          />
          <Input
            label="Uses Per Customer"
            type="number"
            min="0"
            placeholder="Unlimited"
            hint="0 for unlimited"
            error={errors.usage_per_user?.message}
            {...register('usage_per_user', { valueAsNumber: true })}
          />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Valid From"
            type="date"
            error={errors.valid_from?.message}
            required
            {...register('valid_from')}
          />
          <Input
            label="Valid Until"
            type="date"
            error={errors.valid_until?.message}
            required
            {...register('valid_until')}
          />
        </div>

        <div className="pt-2 border-t border-gray-200">
          <p className="text-sm font-medium text-gray-900 mb-1">Scope</p>
          <p className="text-xs text-gray-500 mb-3">
            Leave empty to apply to everything. An include list requires at least one matching item
            in the order; anything excluded refuses the coupon outright.
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <MultiSelect
              label="Include Categories"
              options={categoryOptions}
              {...register('applicable_categories')}
            />
            <MultiSelect
              label="Exclude Categories"
              options={categoryOptions}
              {...register('excluded_categories')}
            />
            <MultiSelect
              label="Include Products"
              options={productOptions}
              {...register('applicable_products')}
            />
            <MultiSelect
              label="Exclude Products"
              options={productOptions}
              {...register('excluded_products')}
            />
          </div>
        </div>

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
