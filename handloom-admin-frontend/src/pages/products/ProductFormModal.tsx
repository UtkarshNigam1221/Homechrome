import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { artisansApi, categoriesApi, getErrorMessage, productsApi } from '../../api';
import { Button, ImageUpload, Input, Modal, Select } from '../../components/common';
import type {
  Artisan,
  Category,
  CategoryAttribute,
  CreateProductRequest,
  Product,
} from '../../types';

const productSchema = z.object({
  name: z.string().min(1, 'Name is required').max(200, 'Name must be less than 200 characters'),
  sku: z.string().min(1, 'SKU is required').max(50, 'SKU must be less than 50 characters'),
  description: z.string().optional(),
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
  initial_stock: z.number().min(0, 'Initial stock must be positive').optional(),
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

  // Dynamic category attribute values (managed outside of Zod since schema is dynamic)
  const [attributeValues, setAttributeValues] = useState<Record<string, unknown>>({});

  // Fetch categories (flat list - no hierarchy)
  const { data: categoriesData } = useQuery({
    queryKey: ['categories-list'],
    queryFn: () => categoriesApi.list({ limit: 100 }),
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
      initial_stock: 0,
      low_stock_threshold: 5,
      status: 'DRAFT',
      tags: '',
      images: [],
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
        // Restore existing attribute values when editing
        setAttributeValues(product.attributes || {});
      } else {
        reset({
          name: '',
          sku: '',
          description: '',
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
          initial_stock: 0,
          low_stock_threshold: 5,
          status: 'DRAFT',
          tags: '',
          images: [],
        });
        setAttributeValues({});
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

  // Handle attribute value changes
  const handleAttributeChange = (attrName: string, value: unknown) => {
    setAttributeValues((prev) => ({ ...prev, [attrName]: value }));
  };

  // Handle multi-select toggle
  const handleMultiSelectToggle = (attrName: string, optionValue: string) => {
    setAttributeValues((prev) => {
      const current = (prev[attrName] as string[]) || [];
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
      toast.error(`Please fill in required attributes: ${missingRequired.map((a) => a.label).join(', ')}`);
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
      initial_stock: data.initial_stock || 0,
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
      attributes: Object.keys(cleanAttributes).length > 0 ? cleanAttributes : undefined,
    };

    if (isEditing && product?.id) {
      // SKU is immutable after creation — strip it from update payload
      const { sku: _, ...updateData } = requestData;
      updateMutation.mutate({ id: product.id, data: updateData });
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

  // Render a dynamic form field based on attribute type
  const renderAttributeField = (attr: CategoryAttribute) => {
    const value = attributeValues[attr.name];

    switch (attr.type) {
      case 'TEXT':
        return (
          <Input
            key={attr.name}
            label={attr.label}
            placeholder={`Enter ${attr.label.toLowerCase()}`}
            value={(value as string) || ''}
            onChange={(e) => handleAttributeChange(attr.name, e.target.value)}
            required={attr.required}
          />
        );

      case 'NUMBER':
        return (
          <Input
            key={attr.name}
            label={attr.label}
            type="number"
            step="any"
            placeholder={`Enter ${attr.label.toLowerCase()}`}
            value={value !== undefined && value !== null ? String(value) : ''}
            onChange={(e) =>
              handleAttributeChange(attr.name, e.target.value ? Number(e.target.value) : '')
            }
            required={attr.required}
          />
        );

      case 'SELECT':
        return (
          <Select
            key={attr.name}
            label={attr.label}
            options={
              attr.options?.map((opt) => ({
                value: opt.value,
                label: opt.label,
              })) || []
            }
            placeholder={`Select ${attr.label.toLowerCase()}`}
            value={(value as string) || ''}
            onChange={(e) => handleAttributeChange(attr.name, e.target.value)}
            required={attr.required}
          />
        );

      case 'MULTI_SELECT':
        return (
          <div key={attr.name}>
            <label className="label">
              {attr.label}
              {attr.required && <span className="text-red-500 ml-1">*</span>}
            </label>
            <div className="mt-1 space-y-2 p-3 border border-gray-200 rounded-lg max-h-40 overflow-y-auto">
              {attr.options && attr.options.length > 0 ? (
                attr.options.map((opt) => {
                  const selectedValues = (value as string[]) || [];
                  const isSelected = selectedValues.includes(opt.value);
                  return (
                    <label
                      key={opt.value}
                      className="flex items-center gap-2 cursor-pointer"
                    >
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => handleMultiSelectToggle(attr.name, opt.value)}
                        className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                      />
                      <span className="text-sm text-gray-700">{opt.label}</span>
                    </label>
                  );
                })
              ) : (
                <p className="text-sm text-gray-500">No options available</p>
              )}
            </div>
          </div>
        );

      case 'BOOLEAN':
        return (
          <div key={attr.name} className="flex items-center gap-3 pt-6">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={!!value}
                onChange={(e) => handleAttributeChange(attr.name, e.target.checked)}
                className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
              />
              <span className="text-sm font-medium text-gray-700">
                {attr.label}
                {attr.required && <span className="text-red-500 ml-1">*</span>}
              </span>
            </label>
          </div>
        );

      default:
        return (
          <Input
            key={attr.name}
            label={attr.label}
            placeholder={`Enter ${attr.label.toLowerCase()}`}
            value={(value as string) || ''}
            onChange={(e) => handleAttributeChange(attr.name, e.target.value)}
            required={attr.required}
          />
        );
    }
  };

  const categories = extractItems<Category>(categoriesData, 'categories');
  const artisans = extractItems<Artisan>(artisansData, 'artisans');

  const categoryOptions = getCategoryOptions(categories);

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
              disabled={isEditing}
              hint={isEditing ? 'SKU cannot be changed after creation' : undefined}
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
          </div>
        </div>

        {/* Category Attributes - shown dynamically when a category is selected */}
        {selectedCategoryId && categoryAttributes.length > 0 && (
          <div>
            <h3 className="text-sm font-medium text-gray-700 mb-1">Category Attributes</h3>
            <p className="text-xs text-gray-500 mb-3">
              These attributes are specific to the selected category
            </p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
              {categoryAttributes
                .sort((a, b) => a.display_order - b.display_order)
                .map((attr) => renderAttributeField(attr))}
            </div>
          </div>
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
