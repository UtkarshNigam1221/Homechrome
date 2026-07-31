import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useMemo, useRef, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { categoriesApi } from '@/features/categories/api';
import type { Category, CategoryAttribute } from '@/features/categories/types';
import { productsApi } from '@/features/products/api';
import { getErrorMessage } from '@/shared/api/client';
import { Button, ImageUpload, Input, Modal, Select } from '@/shared/components/ui';

import type { CreateProductRequest, Product } from '../../types';
import { AttributeFields } from './AttributeFields';

// The SKU pattern applies on create only: the field is locked when editing,
// and legacy products with nonconforming SKUs must stay editable.
const makeProductSchema = (isEditing: boolean) =>
  z.object({
    name: z.string().min(1, 'Name is required').max(200, 'Name must be less than 200 characters'),
    sku: isEditing
      ? z.string()
      : z
          .string()
          .min(1, 'SKU is required')
          .max(50, 'SKU must be less than 50 characters')
          .regex(/^[A-Za-z0-9-]+$/, 'Letters, numbers and dashes only — no spaces'),
    description: z.string().optional(),
    category_id: z.string().min(1, 'Category is required'),
    base_price: z.number().min(0, 'Base price must be positive'),
    selling_price: z.number().min(0, 'Selling price must be positive'),
    cost_price: z.number().min(0, 'Cost price must be positive').optional(),
    material: z.string().optional(),
    weave_type: z.string().optional(),
    origin: z.string().optional(),
    craft_type: z.string().optional(),
    weight: z.number().min(0).optional(),
    length: z.number().min(0).optional(),
    width: z.number().min(0).optional(),
    height: z.number().min(0).optional(),
    dimension_unit: z.string().optional(),
    initial_stock: z.number().min(0, 'Initial stock must be positive').optional(),
    stock_quantity: z.number().min(0, 'Stock must be positive').optional(),
    low_stock_threshold: z.number().min(0, 'Threshold must be positive'),
    status: z.enum(['ACTIVE', 'INACTIVE', 'DRAFT']),
    tags: z.string().optional(),
    images: z.array(z.string()).optional(),
    video_url: z.string().optional().or(z.literal('')),
    video_poster_url: z.string().optional().or(z.literal('')),
  });

type ProductFormData = z.infer<ReturnType<typeof makeProductSchema>>;

interface ProductFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  product?: Product | null;
}

export function ProductFormModal({ isOpen, onClose, product }: ProductFormModalProps) {
  const queryClient = useQueryClient();
  const isEditing = !!product?.id;

  // Dynamic category attribute values (managed outside of Zod since schema is dynamic)
  const [attributeValues, setAttributeValues] = useState<Record<string, unknown>>({});

  // Latches once the user has ever typed in the SKU field. dirtyFields alone
  // is not enough: clearing the field back to '' un-dirties it and would
  // re-arm the auto-generator over a value the user is retyping.
  const skuEdited = useRef(false);

  const schema = useMemo(() => makeProductSchema(!!product?.id), [product?.id]);

  // Fetch categories (flat list - no hierarchy)
  const { data: categoriesData } = useQuery({
    queryKey: ['categories-list', { limit: 100 }],
    queryFn: () => categoriesApi.list({ limit: 100 }),
    enabled: isOpen,
  });

  // Fetch full product detail to get attributes (list API may omit them)
  const { data: fullProduct } = useQuery({
    queryKey: ['product-detail', product?.id],
    queryFn: () => {
      if (!product?.id) throw new Error('product id required');
      return productsApi.get(product.id);
    },
    enabled: isOpen && !!product?.id,
  });

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    control,
    formState: { errors, dirtyFields },
  } = useForm<ProductFormData>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      sku: '',
      description: '',
      category_id: '',

      base_price: 0,
      selling_price: 0,
      cost_price: 0,
      material: '',
      weave_type: '',
      origin: '',
      craft_type: '',
      weight: 0,
      length: 0,
      width: 0,
      height: 0,
      dimension_unit: 'cm',
      initial_stock: 0,
      stock_quantity: 0,
      low_stock_threshold: 5,
      status: 'DRAFT',
      tags: '',
      images: [],
      video_url: '',
      video_poster_url: '',
    },
  });

  const name = watch('name');
  const selectedCategoryId = watch('category_id');

  // Fetch category attributes when a category is selected
  const { data: categoryAttributesData } = useQuery({
    queryKey: ['category-attributes', selectedCategoryId],
    queryFn: () => categoriesApi.getAttributes(selectedCategoryId),
    enabled: !!selectedCategoryId && isOpen,
  });

  // Extract category attributes from the response
  const categoryAttributes: CategoryAttribute[] = selectedCategoryId
    ? [...(categoryAttributesData?.own_attributes || [])]
    : [];

  // Reset attribute values when category changes (but not on initial load for editing)
  useEffect(() => {
    if (!isEditing && selectedCategoryId) {
      setAttributeValues({});
    }
  }, [selectedCategoryId, isEditing]);

  // Auto-generate SKU from name, but only until the user types their own —
  // a manually entered design code (e.g. DBK2228) must never be overwritten.
  useEffect(() => {
    if (dirtyFields.sku) {
      skuEdited.current = true;
    }
    if (!isEditing && name && !skuEdited.current) {
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
  }, [name, isEditing, setValue, dirtyFields.sku]);

  // Reset form when modal opens/closes or product changes
  useEffect(() => {
    if (isOpen) {
      skuEdited.current = isEditing; // fresh create form re-arms auto-generation
      if (product?.id) {
        reset({
          name: product.name,
          sku: product.sku,
          description: product.description || '',
          category_id: product.category_id,
          base_price: product.base_price / 100,
          selling_price: product.selling_price / 100,
          cost_price: product.cost_price ? product.cost_price / 100 : 0,
          material: product.material || '',
          weave_type: product.weave_type || '',
          origin: product.origin || '',
          craft_type: product.craft_type || '',
          weight: product.weight || 0,
          length: product.dimensions?.length || 0,
          width: product.dimensions?.width || 0,
          height: product.dimensions?.height || 0,
          dimension_unit: product.dimensions?.unit || 'cm',
          stock_quantity: product.quantity ?? 0,
          low_stock_threshold: product.low_stock_threshold,
          status: product.status,
          tags: product.tags?.join(', ') || '',
          images: product.images?.map((img) => img.url) || [],
          video_url: product.video_url || '',
          video_poster_url: product.video_poster_url || '',
        });
      } else {
        reset({
          name: '',
          sku: '',
          description: '',
          category_id: '',

          base_price: 0,
          selling_price: 0,
          cost_price: 0,
          material: '',
          weave_type: '',
          origin: '',
          craft_type: '',
          weight: 0,
          length: 0,
          width: 0,
          height: 0,
          dimension_unit: 'cm',
          initial_stock: 0,
          low_stock_threshold: 5,
          status: 'DRAFT',
          tags: '',
          images: [],
          video_url: '',
          video_poster_url: '',
        });
      }
    }
  }, [isOpen, product, reset, isEditing]);

  // Hydrate dynamic attribute values when editing. Tracks both the list-level
  // product (immediate) and the full product detail (async, includes attributes
  // map). Hardcoded fields (material, color, weave_type, origin, craft_type)
  // are also folded in so category attributes that overlap with those slots
  // pre-fill from the dedicated product field when the backend strips them
  // from the attributes map.
  useEffect(() => {
    if (!isOpen || !product?.id) {
      setAttributeValues({});
      return;
    }
    const src = (fullProduct ?? product) as Product;
    const hardcoded: Record<string, unknown> = {};
    if (src.material) hardcoded.material = src.material;
    if (src.weave_type) hardcoded.weave_type = src.weave_type;
    if (src.origin) hardcoded.origin = src.origin;
    if (src.craft_type) hardcoded.craft_type = src.craft_type;
    setAttributeValues({ ...hardcoded, ...(src.attributes || {}) });
  }, [isOpen, product, fullProduct]);

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
    mutationFn: async ({
      id,
      data,
      newStockQty,
    }: {
      id: string;
      data: Partial<CreateProductRequest>;
      newStockQty?: number;
    }) => {
      const result = await productsApi.update(id, data);
      // If stock quantity changed, adjust it via inventory endpoint
      if (newStockQty !== undefined && product && newStockQty !== product.quantity) {
        await productsApi.adjustStock(id, newStockQty, 'Stock adjusted via product edit');
      }
      return result;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['products-inventory'] });
      toast.success('Product updated successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Handle attribute value changes
  const handleAttributeChange = (attrName: string, value: unknown) => {
    setAttributeValues((prev) => ({ ...prev, [attrName]: value }));
  };

  // Handle multi-select toggle
  const handleMultiSelectToggle = (attrName: string, optionValue: string) => {
    setAttributeValues((prev) => {
      const raw = prev[attrName];
      // Normalize: single-value attributes load as a plain string.
      const current = Array.isArray(raw) ? (raw as string[]) : raw ? [String(raw)] : [];
      const newValues = current.includes(optionValue)
        ? current.filter((v) => v !== optionValue)
        : [...current, optionValue];
      return { ...prev, [attrName]: newValues };
    });
  };

  const onSubmit = (data: ProductFormData) => {
    // Validate required attributes
    const missingRequired = categoryAttributes
      .filter((attr) => attr.required)
      .filter((attr) => {
        const val = attributeValues[attr.name];
        if (val === undefined || val === null || val === '') return true;
        if (Array.isArray(val) && val.length === 0) return true;
        return false;
      });

    if (missingRequired.length > 0) {
      toast.error(
        `Please fill in required attributes: ${missingRequired.map((a) => a.label).join(', ')}`
      );
      return;
    }

    // Build clean attributes object (only include non-empty values)
    const cleanAttributes: Record<string, unknown> = {};
    for (const attr of categoryAttributes) {
      const val = attributeValues[attr.name];
      if (val !== undefined && val !== null && val !== '') {
        if (Array.isArray(val) && val.length === 0) continue;
        cleanAttributes[attr.name] = val;
      }
    }

    const requestData: CreateProductRequest = {
      name: data.name,
      sku: data.sku,
      description: data.description || undefined,
      category_id: data.category_id,
      base_price: Math.round(data.base_price * 100),
      selling_price: Math.round(data.selling_price * 100),
      cost_price: data.cost_price ? Math.round(data.cost_price * 100) : undefined,
      material: data.material || undefined,
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
      initial_stock: data.initial_stock || 0,
      low_stock_threshold: data.low_stock_threshold,
      status: data.status,
      tags: data.tags
        ? data.tags
            .split(',')
            .map((t) => t.trim())
            .filter(Boolean)
        : undefined,
      images: (data.images || []).map((url, index) => ({
        url,
        alt_text: data.name,
        is_primary: index === 0,
        sort_order: index,
      })),
      video_url: data.video_url ?? '',
      video_poster_url: data.video_poster_url ?? '',
      attributes: Object.keys(cleanAttributes).length > 0 ? cleanAttributes : undefined,
    };

    if (isEditing && product?.id) {
      // SKU is immutable after creation — strip it from update payload
      const { sku: _, ...updateData } = requestData;
      updateMutation.mutate({ id: product.id, data: updateData, newStockQty: data.stock_quantity });
    } else {
      createMutation.mutate(requestData);
    }
  };

  // Convert categories to select options (flat list - no hierarchy)
  const getCategoryOptions = (cats: Category[]): { value: string; label: string }[] => {
    return cats.map((cat) => ({
      value: cat.id,
      label: cat.name,
    }));
  };

  const categories = categoriesData?.items ?? [];

  const categoryOptions = getCategoryOptions(categories);

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
              placeholder="Design code, e.g. DBK2228"
              error={errors.sku?.message}
              required
              disabled={isEditing}
              hint={
                isEditing
                  ? 'SKU cannot be changed after creation'
                  : 'Type the supplier design code — auto-filled from name until you do'
              }
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

            <div className="md:col-span-2">
              <Controller
                name="video_url"
                control={control}
                render={({ field }) => (
                  <ImageUpload
                    label="Product Video (optional)"
                    value={field.value || ''}
                    onChange={(value) =>
                      field.onChange(Array.isArray(value) ? value[0] || '' : value)
                    }
                    accept="video/mp4"
                    maxSizeMB={50}
                    hint="MP4 only, up to 50MB. One video per product."
                    error={errors.video_url?.message}
                  />
                )}
              />
            </div>

            <div className="md:col-span-2">
              <Controller
                name="video_poster_url"
                control={control}
                render={({ field }) => (
                  <ImageUpload
                    label="Video Poster Image (optional)"
                    value={field.value || ''}
                    onChange={(value) =>
                      field.onChange(Array.isArray(value) ? value[0] || '' : value)
                    }
                    accept="image/*"
                    maxSizeMB={2}
                    hint="Thumbnail shown before the video plays. Recommended 16:9."
                    error={errors.video_poster_url?.message}
                  />
                )}
              />
            </div>
          </div>
        </div>

        {/* Category Attributes - shown dynamically when a category is selected */}
        {selectedCategoryId && categoryAttributes.length > 0 && (
          <AttributeFields
            attributes={categoryAttributes}
            attributeValues={attributeValues}
            onAttributeChange={handleAttributeChange}
            onMultiSelectToggle={handleMultiSelectToggle}
          />
        )}

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
            {!isEditing && (
              <Input
                label="Initial Stock"
                type="number"
                min="0"
                placeholder="0"
                hint="Quantity to add on creation"
                error={errors.initial_stock?.message}
                {...register('initial_stock', { valueAsNumber: true })}
              />
            )}

            {isEditing && (
              <>
                <Input
                  label="Stock Quantity"
                  type="number"
                  min="0"
                  placeholder="0"
                  hint={`Available: ${product?.available_qty ?? 0} · Reserved: ${product?.reserved_qty ?? 0}`}
                  error={errors.stock_quantity?.message}
                  {...register('stock_quantity', { valueAsNumber: true })}
                />
              </>
            )}

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
