import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { categoriesApi } from '@/features/categories/api';
import type { Category } from '@/features/categories/types';
import { pricingApi } from '@/features/pricing/api';
import { Button, Input, Modal, Select } from '@/shared/components/ui';
import { useFormMutation } from '@/shared/hooks';

import type { PricingRule, PricingUnit } from '../types';

const pricingRuleSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be less than 100 characters'),
  description: z.string().optional(),
  scope_type: z.enum(['GLOBAL', 'CATEGORY', 'SUBCATEGORY', 'PRODUCT', 'MATERIAL']),
  category_id: z.string().optional(),
  pricing_type: z.enum(['AREA_BASED', 'LENGTH_BASED', 'FIXED', 'TIERED', 'FORMULA']),
  base_price: z.number().min(0, 'Base price must be positive'),
  price_per_unit: z.number().min(0).optional(),
  unit: z.string().optional(),
  min_area: z.number().min(0).optional(),
  max_area: z.number().min(0).optional(),
  priority: z.number().min(0, 'Priority must be positive'),
  is_active: z.boolean(),
});

type PricingRuleFormData = z.infer<typeof pricingRuleSchema>;

interface PricingRuleFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  rule?: PricingRule | null;
}

export function PricingRuleFormModal({ isOpen, onClose, rule }: PricingRuleFormModalProps) {
  const isEditing = !!rule?.id;

  // Fetch categories
  const { data: categoriesData } = useQuery({
    queryKey: ['categories-list'],
    queryFn: () => categoriesApi.list({ limit: 100 }),
    enabled: isOpen,
  });

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<PricingRuleFormData>({
    resolver: zodResolver(pricingRuleSchema),
    defaultValues: {
      name: '',
      description: '',
      scope_type: 'GLOBAL',
      category_id: '',
      pricing_type: 'FIXED',
      base_price: 0,
      price_per_unit: 0,
      unit: 'SQ_INCH',
      min_area: 0,
      max_area: 0,
      priority: 1,
      is_active: true,
    },
  });

  const scopeType = watch('scope_type');
  const pricingType = watch('pricing_type');

  // Reset form when modal opens/closes or rule changes
  useEffect(() => {
    if (isOpen) {
      if (rule?.id) {
        reset({
          name: rule.name,
          description: rule.description || '',
          scope_type: rule.scope_type,
          category_id: rule.category_id || '',
          pricing_type: rule.pricing_type,
          base_price: rule.base_price / 100,
          price_per_unit: rule.price_per_unit ? rule.price_per_unit / 100 : 0,
          unit: rule.unit || 'SQ_INCH',
          min_area: rule.min_area || 0,
          max_area: rule.max_area || 0,
          priority: rule.priority,
          is_active: rule.is_active,
        });
      } else {
        reset({
          name: '',
          description: '',
          scope_type: 'GLOBAL',
          category_id: '',
          pricing_type: 'FIXED',
          base_price: 0,
          price_per_unit: 0,
          unit: 'SQ_INCH',
          min_area: 0,
          max_area: 0,
          priority: 1,
          is_active: true,
        });
      }
    }
  }, [isOpen, rule, reset]);

  const { isLoading, onSubmit: submitMutation } = useFormMutation<
    Partial<PricingRule>,
    Partial<PricingRule>
  >({
    queryKey: 'pricing-rules',
    createFn: pricingApi.createRule,
    updateFn: pricingApi.updateRule,
    entityName: 'Pricing rule',
    onSuccess: onClose,
  });

  const onSubmit = (data: PricingRuleFormData) => {
    const requestData: Partial<PricingRule> = {
      name: data.name,
      description: data.description || undefined,
      scope_type: data.scope_type,
      category_id: data.category_id || undefined,
      pricing_type: data.pricing_type,
      base_price: Math.round(data.base_price * 100),
      price_per_unit: data.price_per_unit ? Math.round(data.price_per_unit * 100) : undefined,
      unit: (data.unit as PricingUnit) || undefined,
      min_area: data.min_area || undefined,
      max_area: data.max_area || undefined,
      priority: data.priority,
      is_active: data.is_active,
    };

    submitMutation(rule?.id, requestData, requestData);
  };

  // Map categories to select options (flat list, no hierarchy)
  const flattenCategories = (cats: Category[]): { value: string; label: string }[] => {
    return cats.map((cat) => ({
      value: cat.id,
      label: cat.name,
    }));
  };

  const categories = categoriesData?.items ?? [];
  const categoryOptions = [
    { value: '', label: 'Select a category' },
    ...flattenCategories(categories),
  ];

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Pricing Rule' : 'Create Pricing Rule'}
      size="lg"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Basic Information */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Basic Information</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              label="Rule Name"
              placeholder="e.g., Bedsheet Area Pricing"
              error={errors.name?.message}
              required
              {...register('name')}
            />

            <Input
              label="Priority"
              type="number"
              min="0"
              placeholder="1"
              hint="Higher priority rules are applied first"
              error={errors.priority?.message}
              required
              {...register('priority', { valueAsNumber: true })}
            />

            <div className="md:col-span-2">
              <label className="label">Description</label>
              <textarea
                className="input min-h-[60px]"
                placeholder="Describe when this rule should be applied..."
                {...register('description')}
              />
            </div>
          </div>
        </div>

        {/* Scope */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Scope</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Select
              label="Scope Type"
              options={[
                { value: 'GLOBAL', label: 'Global (All Products)' },
                { value: 'CATEGORY', label: 'Category' },
                { value: 'SUBCATEGORY', label: 'Subcategory' },
                { value: 'PRODUCT', label: 'Product' },
                { value: 'MATERIAL', label: 'Material' },
              ]}
              error={errors.scope_type?.message}
              required
              {...register('scope_type')}
            />

            {(scopeType === 'CATEGORY' || scopeType === 'SUBCATEGORY') && (
              <Select
                label="Category"
                options={categoryOptions}
                error={errors.category_id?.message}
                {...register('category_id')}
              />
            )}
          </div>
        </div>

        {/* Pricing Configuration */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Pricing Configuration</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Select
              label="Pricing Type"
              options={[
                { value: 'FIXED', label: 'Fixed Price' },
                { value: 'AREA_BASED', label: 'Area Based (sq. units)' },
                { value: 'LENGTH_BASED', label: 'Length Based' },
                { value: 'TIERED', label: 'Tiered Pricing' },
                { value: 'FORMULA', label: 'Custom Formula' },
              ]}
              error={errors.pricing_type?.message}
              required
              {...register('pricing_type')}
            />

            <Input
              label="Base Price (₹)"
              type="number"
              step="0.01"
              min="0"
              placeholder="0"
              error={errors.base_price?.message}
              required
              {...register('base_price', { valueAsNumber: true })}
            />

            {(pricingType === 'AREA_BASED' || pricingType === 'LENGTH_BASED') && (
              <>
                <Input
                  label="Price Per Unit (₹)"
                  type="number"
                  step="0.01"
                  min="0"
                  placeholder="0"
                  error={errors.price_per_unit?.message}
                  {...register('price_per_unit', { valueAsNumber: true })}
                />

                <Select
                  label="Unit"
                  options={[
                    { value: 'SQ_INCH', label: 'Square Inch' },
                    { value: 'SQ_FOOT', label: 'Square Foot' },
                    { value: 'SQ_CM', label: 'Square CM' },
                    { value: 'SQ_METER', label: 'Square Meter' },
                    { value: 'INCH', label: 'Inch' },
                    { value: 'FOOT', label: 'Foot' },
                    { value: 'CM', label: 'Centimeter' },
                    { value: 'METER', label: 'Meter' },
                  ]}
                  {...register('unit')}
                />

                <Input
                  label="Minimum Area"
                  type="number"
                  step="1"
                  min="0"
                  placeholder="No minimum"
                  {...register('min_area', { valueAsNumber: true })}
                />

                <Input
                  label="Maximum Area"
                  type="number"
                  step="1"
                  min="0"
                  placeholder="No maximum"
                  {...register('max_area', { valueAsNumber: true })}
                />
              </>
            )}
          </div>
        </div>

        {/* Status */}
        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="is_active"
            className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            {...register('is_active')}
          />
          <label htmlFor="is_active" className="text-sm text-gray-700">
            Rule is active and will be applied to matching products
          </label>
        </div>

        {/* Form Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Update Rule' : 'Create Rule'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
