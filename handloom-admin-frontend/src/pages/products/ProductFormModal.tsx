import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { artisansApi, categoriesApi, designsApi, getErrorMessage, productsApi } from '../../api';
import { Button, ImageUpload, Input, Modal, Select } from '../../components/common';
import type { Artisan, Category, CreateProductRequest, Design, Product } from '../../types';

const productSchema = z.object({
  name: z.string().min(1, 'Name is required').max(200, 'Name must be less than 200 characters'),
  sku: z.string().min(1, 'SKU is required').max(50, 'SKU must be less than 50 characters'),
  description: z.string().optional(),
  design_id: z.string().min(1, 'Design is required'),
  category_id: z.string().min(1, 'Category is required'),
  artisan_id: z.string().optional(),
  base_price: z.number().min(0, 'Base price must be positive'),
  selling_price: z.number().min(0, 'Selling price must be positive'),
  cost_price: z.number().min(0, 'Cost price must be positive').optional(),
  material: z.string().optional(),
  color: z.string().optional(),
  weave_type: z.string().optional(),
  origin: z.string().optional(),
  craft_type: z.string().optional(),
  weight: z.number().min(0).optional(),
  length: z.number().min(0).optional(),
  width: z.number().min(0).optional(),
  height: z.number().min(0).optional(),
  dimension_unit: z.string().optional(),
  low_stock_threshold: z.number().min(0, 'Threshold must be positive'),
  status: z.enum(['ACTIVE', 'INACTIVE', 'DRAFT']),
  tags: z.string().optional(),
  images: z.array(z.string()).optional(),
});

type ProductFormData = z.infer<typeof productSchema>;

interface ProductFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  product?: Product | null;
}

export function ProductFormModal({ isOpen, onClose, product }: ProductFormModalProps) {
  const queryClient = useQueryClient();
  const isEditing = !!product?.id;

  // Fetch categories
  const { data: categoriesData } = useQuery({
    queryKey: ['categories-list'],
    queryFn: () => categoriesApi.list({ limit: 100 }),
    enabled: isOpen,
  });

  // Fetch designs
  const { data: designsData } = useQuery({
    queryKey: ['designs-list'],
    queryFn: () => designsApi.list({ limit: 100 }),
    enabled: isOpen,
  });

  // Fetch artisans
  const { data: artisansData } = useQuery({
    queryKey: ['artisans-list'],
    queryFn: () => artisansApi.list({ limit: 100 }),
    enabled: isOpen,
  });

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    control,
    formState: { errors },
  } = useForm<ProductFormData>({
    resolver: zodResolver(productSchema),
    defaultValues: {
      name: '',
      sku: '',
      description: '',
      design_id: '',
      category_id: '',
      artisan_id: '',
      base_price: 0,
      selling_price: 0,
      cost_price: 0,
      material: '',
      color: '',
      weave_type: '',
      origin: '',
      craft_type: '',
      weight: 0,
      length: 0,
      width: 0,
      height: 0,
      dimension_unit: 'cm',
      low_stock_threshold: 5,
      status: 'DRAFT',
      tags: '',
      images: [],
    },
  });

  const name = watch('name');

  // Auto-generate SKU from name
  useEffect(() => {
    if (!isEditing && name) {
      const sku =
        name
          .toUpperCase()
          .replace(/[^A-Z0-9\s]/g, '')
          .replace(/\s+/g, '-')
          .slice(0, 20) +
        '-' +
        Math.random().toString(36).substring(2, 6).toUpperCase();
      setValue('sku', sku);
    }
  }, [name, isEditing, setValue]);

  // Reset form when modal opens/closes or product changes
  useEffect(() => {
    if (isOpen) {
      if (product?.id) {
        reset({
          name: product.name,
          sku: product.sku,
          description: product.description || '',
          design_id: product.design_id,
          category_id: product.category_id,
          artisan_id: product.artisan_id || '',
          base_price: product.base_price / 100,
          selling_price: product.selling_price / 100,
          cost_price: product.cost_price ? product.cost_price / 100 : 0,
          material: product.material || '',
          color: product.color || '',
          weave_type: product.weave_type || '',
          origin: product.origin || '',
          craft_type: product.craft_type || '',
          weight: product.weight || 0,
          length: product.dimensions?.length || 0,
          width: product.dimensions?.width || 0,
          height: product.dimensions?.height || 0,
          dimension_unit: product.dimensions?.unit || 'cm',
          low_stock_threshold: product.low_stock_threshold,
          status: product.status,
          tags: product.tags?.join(', ') || '',
          images: product.images?.map((img) => img.url) || [],
        });
      } else {
        reset({
          name: '',
          sku: '',
          description: '',
          design_id: '',
          category_id: '',
          artisan_id: '',
          base_price: 0,
          selling_price: 0,
          cost_price: 0,
          material: '',
          color: '',
          weave_type: '',
          origin: '',
          craft_type: '',
          weight: 0,
          length: 0,
          width: 0,
          height: 0,
          dimension_unit: 'cm',
          low_stock_threshold: 5,
          status: 'DRAFT',
          tags: '',
          images: [],
        });
      }
    }
  }, [isOpen, product, reset]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateProductRequest) => productsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      toast.success('Product created successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateProductRequest> }) =>
      productsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      toast.success('Product updated successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: ProductFormData) => {
    const requestData: CreateProductRequest = {
      name: data.name,
      sku: data.sku,
      design_id: data.design_id,
      category_id: data.category_id,
      artisan_id: data.artisan_id || undefined,
      base_price: Math.round(data.base_price * 100),
      selling_price: Math.round(data.selling_price * 100),
      cost_price: data.cost_price ? Math.round(data.cost_price * 100) : undefined,
      material: data.material || undefined,
      color: data.color || undefined,
      weave_type: data.weave_type || undefined,
      origin: data.origin || undefined,
      craft_type: data.craft_type || undefined,
      weight: data.weight || undefined,
      dimensions:
        data.length || data.width
          ? {
              length: data.length || 0,
              width: data.width || 0,
              height: data.height,
              unit: data.dimension_unit || 'cm',
            }
          : undefined,
      low_stock_threshold: data.low_stock_threshold,
      status: data.status,
      tags: data.tags
        ? data.tags
            .split(',')
            .map((t) => t.trim())
            .filter(Boolean)
        : undefined,
      images:
        data.images && data.images.length > 0
          ? data.images.map((url, index) => ({
              url,
              alt_text: data.name,
              is_primary: index === 0,
              sort_order: index,
            }))
          : undefined,
    };

    if (isEditing && product?.id) {
      updateMutation.mutate({ id: product.id, data: requestData });
    } else {
      createMutation.mutate(requestData);
    }
  };

  // Flatten categories for select
  const flattenCategories = (cats: Category[], depth = 0): { value: string; label: string }[] => {
    const result: { value: string; label: string }[] = [];
    cats.forEach((cat) => {
      const prefix = depth > 0 ? '—'.repeat(depth) + ' ' : '';
      result.push({
        value: cat.id,
        label: `${prefix}${cat.name}`,
      });
      if (cat.children && cat.children.length > 0) {
        result.push(...flattenCategories(cat.children, depth + 1));
      }
    });
    return result;
  };

  // Handle various response formats from the API
  const extractItems = <T,>(data: unknown, key?: string): T[] => {
    if (!data) return [];
    if (Array.isArray(data)) return data as T[];
    if (typeof data === 'object' && data !== null) {
      const record = data as Record<string, unknown>;
      if (key && key in record) return record[key] as T[];
      if ('items' in record) return record.items as T[];
      if ('data' in record) return Array.isArray(record.data) ? (record.data as T[]) : [];
    }
    return [];
  };

  const categories = extractItems<Category>(categoriesData, 'categories');
  const designs = extractItems<Design>(designsData, 'designs');
  const artisans = extractItems<Artisan>(artisansData, 'artisans');

  const categoryOptions = flattenCategories(categories);

  const designOptions = designs.map((d) => ({ value: d.id, label: d.name }));

  const artisanOptions = [
    { value: '', label: 'No artisan assigned' },
    ...artisans.map((a) => ({ value: a.id, label: a.name })),
  ];

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Product' : 'Create Product'}
      size="lg"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Basic Information */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Basic Information</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              label="Product Name"
              placeholder="e.g., Handwoven Silk Bedsheet"
              error={errors.name?.message}
              required
              {...register('name')}
            />

            <Input
              label="SKU"
              placeholder="e.g., SILK-BS-001"
              error={errors.sku?.message}
              required
              {...register('sku')}
            />

            <Select
              label="Category"
              options={categoryOptions}
              placeholder="Select a category"
              error={errors.category_id?.message}
              required
              {...register('category_id')}
            />

            <Select
              label="Design"
              options={designOptions}
              placeholder="Select a design"
              error={errors.design_id?.message}
              required
              {...register('design_id')}
            />

            <div className="md:col-span-2">
              <label className="label">Description</label>
              <textarea
                className="input min-h-[80px]"
                placeholder="Product description..."
                {...register('description')}
              />
            </div>

            <div className="md:col-span-2">
              <Controller
                name="images"
                control={control}
                render={({ field }) => (
                  <ImageUpload
                    label="Product Images"
                    value={field.value || []}
                    onChange={(value) => field.onChange(Array.isArray(value) ? value : [value])}
                    multiple
                    maxFiles={5}
                    hint="Upload up to 5 product images"
                    error={errors.images?.message}
                  />
                )}
              />
            </div>
          </div>
        </div>

        {/* Pricing */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Pricing (in INR)</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Input
              label="Base Price"
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
              error={errors.base_price?.message}
              required
              {...register('base_price', { valueAsNumber: true })}
            />

            <Input
              label="Selling Price"
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
              error={errors.selling_price?.message}
              required
              {...register('selling_price', { valueAsNumber: true })}
            />

            <Input
              label="Cost Price"
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
              error={errors.cost_price?.message}
              {...register('cost_price', { valueAsNumber: true })}
            />
          </div>
        </div>

        {/* Product Details */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Product Details</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Input label="Material" placeholder="e.g., Silk, Cotton" {...register('material')} />

            <Input label="Color" placeholder="e.g., Red, Blue" {...register('color')} />

            <Input
              label="Weave Type"
              placeholder="e.g., Jacquard, Plain"
              {...register('weave_type')}
            />

            <Input
              label="Origin"
              placeholder="e.g., Varanasi, Kanchipuram"
              {...register('origin')}
            />

            <Input
              label="Craft Type"
              placeholder="e.g., Handloom, Powerloom"
              {...register('craft_type')}
            />

            <Select label="Artisan" options={artisanOptions} {...register('artisan_id')} />
          </div>
        </div>

        {/* Dimensions */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Dimensions & Weight</h3>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <Input
              label="Length"
              type="number"
              step="0.1"
              min="0"
              placeholder="0"
              {...register('length', { valueAsNumber: true })}
            />

            <Input
              label="Width"
              type="number"
              step="0.1"
              min="0"
              placeholder="0"
              {...register('width', { valueAsNumber: true })}
            />

            <Input
              label="Height"
              type="number"
              step="0.1"
              min="0"
              placeholder="0"
              {...register('height', { valueAsNumber: true })}
            />

            <Select
              label="Unit"
              options={[
                { value: 'cm', label: 'Centimeters' },
                { value: 'inch', label: 'Inches' },
                { value: 'm', label: 'Meters' },
                { value: 'ft', label: 'Feet' },
              ]}
              {...register('dimension_unit')}
            />

            <Input
              label="Weight (g)"
              type="number"
              step="1"
              min="0"
              placeholder="0"
              {...register('weight', { valueAsNumber: true })}
            />
          </div>
        </div>

        {/* Inventory & Status */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Inventory & Status</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Input
              label="Low Stock Threshold"
              type="number"
              min="0"
              placeholder="5"
              error={errors.low_stock_threshold?.message}
              {...register('low_stock_threshold', { valueAsNumber: true })}
            />

            <Select
              label="Status"
              options={[
                { value: 'DRAFT', label: 'Draft' },
                { value: 'ACTIVE', label: 'Active' },
                { value: 'INACTIVE', label: 'Inactive' },
              ]}
              required
              {...register('status')}
            />

            <Input
              label="Tags"
              placeholder="e.g., silk, wedding, festive"
              hint="Comma-separated tags"
              {...register('tags')}
            />
          </div>
        </div>

        {/* Form Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Update Product' : 'Create Product'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
