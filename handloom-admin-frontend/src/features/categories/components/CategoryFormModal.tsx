import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2 } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Controller, useFieldArray, useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { categoriesApi } from '@/features/categories/api';
import { getErrorMessage } from '@/shared/api/client';
import { Button, ImageUpload, Input, Modal, Select } from '@/shared/components/ui';

import type { AttributeType, Category, CategoryAttribute, CreateCategoryRequest } from '../types';

const attributeOptionSchema = z.object({
  value: z.string().min(1, 'Value is required'),
  label: z.string().min(1, 'Label is required'),
  surcharge: z.number().optional(),
});

const attributeSchema = z.object({
  name: z
    .string()
    .min(1, 'Name is required')
    .regex(/^[a-z0-9_]+$/, 'Name must be lowercase letters, numbers, and underscores only'),
  label: z.string().min(1, 'Label is required'),
  type: z.enum([
    'SELECT',
    'MULTI_SELECT',
    'TEXT',
    'NUMBER',
    'BOOLEAN',
    'DIMENSION',
    'DIMENSION_RANGE',
  ] as const),
  required: z.boolean(),
  searchable: z.boolean(),
  display_order: z.number(),
  options: z.array(attributeOptionSchema).optional(),
});

const categorySchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be less than 100 characters'),
  description: z.string().optional(),
  image_url: z.string(),
  status: z.enum(['ACTIVE', 'INACTIVE']),
  own_attributes: z.array(attributeSchema).optional(),
});

type CategoryFormData = z.infer<typeof categorySchema>;

interface CategoryFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  category?: Category | null;
}

const ATTRIBUTE_TYPES: { value: AttributeType; label: string }[] = [
  { value: 'SELECT', label: 'Single Select' },
  { value: 'MULTI_SELECT', label: 'Multi Select' },
  { value: 'TEXT', label: 'Text' },
  { value: 'NUMBER', label: 'Number' },
  { value: 'BOOLEAN', label: 'Yes/No' },
];

export function CategoryFormModal({ isOpen, onClose, category }: CategoryFormModalProps) {
  const queryClient = useQueryClient();
  const isEditing = !!category?.id;
  const [activeTab, setActiveTab] = useState<'basic' | 'attributes'>('basic');

  const {
    register,
    handleSubmit,
    control,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = useForm<CategoryFormData>({
    resolver: zodResolver(categorySchema),
    defaultValues: {
      name: '',
      description: '',
      image_url: '',
      status: 'ACTIVE',
      own_attributes: [],
    },
  });

  const {
    fields: attributeFields,
    append: appendAttribute,
    remove: removeAttribute,
  } = useFieldArray({
    control,
    name: 'own_attributes',
  });

  const attributes = watch('own_attributes') || [];

  // Reset form when modal opens/closes or category changes
  useEffect(() => {
    if (isOpen) {
      setActiveTab('basic');
      if (category?.id) {
        reset({
          name: category.name,
          description: category.description || '',
          image_url: category.image_url || '',
          status: category.status,
          own_attributes: category.own_attributes || [],
        });
      } else {
        reset({
          name: '',
          description: '',
          image_url: '',
          status: 'ACTIVE',
          own_attributes: [],
        });
      }
    }
  }, [isOpen, category, reset]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateCategoryRequest) => categoriesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      toast.success('Category created successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateCategoryRequest> }) =>
      categoriesApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories'] });
      toast.success('Category updated successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: CategoryFormData) => {
    const requestData: CreateCategoryRequest & { own_attributes?: CategoryAttribute[] } = {
      name: data.name,
      description: data.description,
      image_url: data.image_url || undefined,
      status: data.status,
      own_attributes: data.own_attributes as CategoryAttribute[],
    };

    if (isEditing && category?.id) {
      updateMutation.mutate({ id: category.id, data: requestData });
    } else {
      createMutation.mutate(requestData);
    }
  };

  const addNewAttribute = () => {
    appendAttribute({
      name: '',
      label: '',
      type: 'SELECT' as AttributeType,
      required: false,
      searchable: false,
      display_order: attributeFields.length,
      options: [],
    });
  };

  const addOptionToAttribute = (attributeIndex: number) => {
    const currentOptions = attributes[attributeIndex]?.options || [];
    setValue(`own_attributes.${attributeIndex}.options`, [
      ...currentOptions,
      { value: '', label: '' },
    ]);
  };

  const removeOptionFromAttribute = (attributeIndex: number, optionIndex: number) => {
    const currentOptions = attributes[attributeIndex]?.options || [];
    setValue(
      `own_attributes.${attributeIndex}.options`,
      currentOptions.filter((_, idx) => idx !== optionIndex)
    );
  };

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Category' : 'Create Category'}
      size="lg"
    >
      {/* Tabs */}
      <div className="flex border-b border-gray-200 mb-4" role="tablist">
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'basic'}
          aria-controls="tabpanel-basic"
          onClick={() => setActiveTab('basic')}
          className={`px-4 py-2 text-sm font-medium border-b-2 ${
            activeTab === 'basic'
              ? 'border-primary-500 text-primary-600'
              : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          Basic Info
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'attributes'}
          aria-controls="tabpanel-attributes"
          onClick={() => setActiveTab('attributes')}
          className={`px-4 py-2 text-sm font-medium border-b-2 ${
            activeTab === 'attributes'
              ? 'border-primary-500 text-primary-600'
              : 'border-transparent text-gray-500 hover:text-gray-700'
          }`}
        >
          Attributes
          {attributeFields.length > 0 && (
            <span className="ml-2 px-2 py-0.5 text-xs bg-gray-100 rounded-full">
              {attributeFields.length}
            </span>
          )}
        </button>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        {activeTab === 'basic' && (
          <div role="tabpanel" id="tabpanel-basic">
            <Input
              label="Name"
              placeholder="e.g., Bedsheets"
              error={errors.name?.message}
              required
              {...register('name')}
            />

            <div>
              <label className="label">Description</label>
              <textarea
                className="input min-h-[80px]"
                placeholder="Optional description for this category"
                {...register('description')}
              />
            </div>

            <Select
              label="Status"
              options={[
                { value: 'ACTIVE', label: 'Active' },
                { value: 'INACTIVE', label: 'Inactive' },
              ]}
              required
              {...register('status')}
            />

            <Controller
              name="image_url"
              control={control}
              render={({ field }) => (
                <ImageUpload
                  label="Category Image"
                  value={field.value || ''}
                  onChange={(value) =>
                    field.onChange(Array.isArray(value) ? value[0] || '' : value)
                  }
                  hint="Upload a category image (optional)"
                />
              )}
            />
          </div>
        )}

        {activeTab === 'attributes' && (
          <div className="space-y-4" role="tabpanel" id="tabpanel-attributes">
            <div className="flex items-center justify-between">
              <p className="text-sm text-gray-600">
                Define attributes for products in this category. Searchable attributes will be
                available as filters in the product list.
              </p>
              <Button type="button" variant="secondary" size="sm" onClick={addNewAttribute}>
                <Plus className="w-4 h-4 mr-1" />
                Add Attribute
              </Button>
            </div>

            {attributeFields.length === 0 ? (
              <div className="text-center py-8 text-gray-500 border border-dashed border-gray-300 rounded-lg">
                <p>No attributes defined</p>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={addNewAttribute}
                  className="mt-2"
                >
                  Add your first attribute
                </Button>
              </div>
            ) : (
              <div className="space-y-4 max-h-[400px] overflow-y-auto">
                {attributeFields.map((field, index) => {
                  const attrType = attributes[index]?.type;
                  const showOptions = attrType === 'SELECT' || attrType === 'MULTI_SELECT';

                  return (
                    <div key={field.id} className="border border-gray-200 rounded-lg p-4 space-y-3">
                      <div className="flex items-start justify-between">
                        <span className="text-sm font-medium text-gray-700">
                          Attribute #{index + 1}
                        </span>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => removeAttribute(index)}
                          className="text-red-600 hover:text-red-700"
                        >
                          <Trash2 className="w-4 h-4" />
                        </Button>
                      </div>

                      <div className="grid grid-cols-2 gap-3">
                        <Input
                          label="Name (Key)"
                          placeholder="e.g., material"
                          hint="Lowercase, no spaces"
                          error={errors.own_attributes?.[index]?.name?.message}
                          {...register(`own_attributes.${index}.name`)}
                        />
                        <Input
                          label="Label"
                          placeholder="e.g., Material"
                          error={errors.own_attributes?.[index]?.label?.message}
                          {...register(`own_attributes.${index}.label`)}
                        />
                      </div>

                      <div className="grid grid-cols-2 gap-3">
                        <Select
                          label="Type"
                          options={ATTRIBUTE_TYPES}
                          {...register(`own_attributes.${index}.type`)}
                        />
                        <Input
                          label="Display Order"
                          type="number"
                          {...register(`own_attributes.${index}.display_order`, {
                            valueAsNumber: true,
                          })}
                        />
                      </div>

                      <div className="flex flex-wrap gap-4">
                        <label className="flex items-center gap-2">
                          <input
                            type="checkbox"
                            className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
                            {...register(`own_attributes.${index}.required`)}
                          />
                          <span className="text-sm text-gray-700">Required</span>
                        </label>
                        <label className="flex items-center gap-2">
                          <input
                            type="checkbox"
                            className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
                            {...register(`own_attributes.${index}.searchable`)}
                          />
                          <span className="text-sm text-gray-700">
                            Searchable (show in filters)
                          </span>
                        </label>
                      </div>

                      {/* Options for SELECT/MULTI_SELECT */}
                      {showOptions && (
                        <div className="mt-3 pt-3 border-t border-gray-100">
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-sm font-medium text-gray-600">Options</span>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => addOptionToAttribute(index)}
                            >
                              <Plus className="w-3 h-3 mr-1" />
                              Add Option
                            </Button>
                          </div>
                          <div className="space-y-2">
                            {(attributes[index]?.options || []).map((_, optIdx) => (
                              <div key={optIdx} className="flex items-center gap-2">
                                <Input
                                  placeholder="Value"
                                  className="flex-1"
                                  {...register(`own_attributes.${index}.options.${optIdx}.value`)}
                                />
                                <Input
                                  placeholder="Label"
                                  className="flex-1"
                                  {...register(`own_attributes.${index}.options.${optIdx}.label`)}
                                />
                                <Button
                                  type="button"
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => removeOptionFromAttribute(index, optIdx)}
                                  className="text-red-500"
                                >
                                  <Trash2 className="w-4 h-4" />
                                </Button>
                              </div>
                            ))}
                            {(attributes[index]?.options?.length || 0) === 0 && (
                              <p className="text-xs text-gray-400 text-center py-2">
                                No options. Add options for users to select from.
                              </p>
                            )}
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Update Category' : 'Create Category'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
