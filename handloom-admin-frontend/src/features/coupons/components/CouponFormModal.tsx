import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { z } from 'zod';

import { couponsApi } from '@/features/coupons/api';
import { customersApi } from '@/features/customers/api';
import { customerDisplayName } from '@/features/customers/lib/displayName';
import type { Customer } from '@/features/customers/types';
import { Button, Input, Modal, Select, type SelectOption } from '@/shared/components/ui';
import { useDebounce, useFormMutation } from '@/shared/hooks';

import {
  type CouponFormValues,
  couponToFormValues,
  defaultCouponFormValues,
  toCreateRequest,
  toUpdateRequest,
} from '../lib/toCreateRequest';
import type { Coupon, CreateCouponRequest, UpdateCouponRequest } from '../types';

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
    description: z.string(),
    type: z.enum(['PERCENTAGE', 'FIXED']),
    value: z.number().gt(0, 'Must be greater than 0'),
    minOrderValue: z.number().min(0),
    maxDiscount: z.number().min(0),
    usageLimit: z.number().min(0),
    usagePerUser: z.number().min(0),
    audience: z.enum(['ALL', 'FIRST_ORDER', 'RETURNING', 'SPECIFIC_CUSTOMER']),
    customerId: z.string(),
    combinesWithOffers: z.boolean(),
    validFrom: z.string().min(1, 'Valid from date is required'),
    noEndDate: z.boolean(),
    expiryDate: z.string(),
    status: z.enum(['ACTIVE', 'INACTIVE', 'EXPIRED']),
  })
  .refine((data) => data.audience !== 'SPECIFIC_CUSTOMER' || data.customerId.trim().length > 0, {
    message: 'Choose a customer for a single-customer coupon',
    path: ['customerId'],
  })
  .refine((data) => data.noEndDate || data.expiryDate.trim().length > 0, {
    message: 'Set an end date, or mark this coupon open-ended',
    path: ['expiryDate'],
  });

const audienceOptions: SelectOption[] = [
  { value: 'ALL', label: 'Everyone' },
  { value: 'FIRST_ORDER', label: 'First order only' },
  { value: 'RETURNING', label: 'Returning customers' },
  { value: 'SPECIFIC_CUSTOMER', label: 'One specific customer' },
];

// A search box plus a Select, matching this codebase's other forms. Keeps the current
// customer visible in the options even before a search would surface them again.
function CustomerPicker({
  value,
  onChange,
  error,
  disabled,
}: {
  value: string;
  onChange: (id: string) => void;
  error?: string;
  disabled?: boolean;
}) {
  const [query, setQuery] = useState('');
  const debouncedQuery = useDebounce(query, 300);

  const { data: results, isFetching } = useQuery({
    queryKey: ['customers-search', debouncedQuery],
    queryFn: () => customersApi.search(debouncedQuery),
    enabled: debouncedQuery.trim().length >= 2,
  });

  const { data: selectedCustomer } = useQuery({
    queryKey: ['customer', value],
    queryFn: () => customersApi.get(value),
    enabled: !!value,
  });

  const label = (c: Customer) => `${customerDisplayName(c)} (${c.phone || c.email})`;

  const options: SelectOption[] = [];
  if (selectedCustomer) {
    options.push({ value: selectedCustomer.id, label: label(selectedCustomer) });
  }
  for (const c of results?.items ?? []) {
    if (c.id === selectedCustomer?.id) continue;
    options.push({ value: c.id, label: label(c) });
  }

  return (
    <div className="space-y-2 rounded-lg border border-gray-200 p-3">
      <Input
        label="Search customers"
        placeholder="Search by name, email, or phone"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        disabled={disabled}
        hint={isFetching ? 'Searching…' : undefined}
      />
      <Select
        label="Customer"
        placeholder={options.length ? 'Select a customer' : 'Type at least 2 characters to search'}
        options={options}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        error={error}
        disabled={disabled}
        required
      />
    </div>
  );
}

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
    control,
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

        {audience === 'SPECIFIC_CUSTOMER' && (
          <Controller
            name="customerId"
            control={control}
            render={({ field, fieldState }) => (
              <CustomerPicker
                value={field.value}
                onChange={field.onChange}
                error={fieldState.error?.message}
                disabled={lockedAfterCreate}
              />
            )}
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
