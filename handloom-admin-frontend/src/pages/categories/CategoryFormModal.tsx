import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { categoriesApi, getErrorMessage } from '../../api';
import { Button, Input, Modal, Select } from '../../components/common';
import type { Category, CreateCategoryRequest } from '../../types';

const categorySchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be less than 100 characters'),
  slug: z
    .string()
    .min(1, 'Slug is required')
    .regex(/^[a-z0-9-]+$/, 'Slug must be lowercase letters, numbers, and hyphens only'),
  description: z.string().optional(),
  parent_id: z.string().optional(),
  status: z.enum(['ACTIVE', 'INACTIVE']),
  allow_custom_dimensions: z.boolean(),
});

type CategoryFormData = z.infer<typeof categorySchema>;

interface CategoryFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  category?: Category | null;
  categories: Category[];
}

export function CategoryFormModal({
  isOpen,
  onClose,
  category,
  categories,
}: CategoryFormModalProps) {
  const queryClient = useQueryClient();
  const isEditing = !!category?.id;

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = useForm<CategoryFormData>({
    resolver: zodResolver(categorySchema),
    defaultValues: {
      name: '',
      slug: '',
      description: '',
      parent_id: '',
      status: 'ACTIVE',
      allow_custom_dimensions: false,
    },
  });

  const name = watch('name');

  // Auto-generate slug from name
  useEffect(() => {
    if (!isEditing && name) {
      const slug = name
        .toLowerCase()
        .replace(/[^a-z0-9\s-]/g, '')
        .replace(/\s+/g, '-')
        .replace(/-+/g, '-')
        .trim();
      setValue('slug', slug);
    }
  }, [name, isEditing, setValue]);

  // Reset form when modal opens/closes or category changes
  useEffect(() => {
    if (isOpen) {
      if (category?.id) {
        reset({
          name: category.name,
          slug: category.slug,
          description: category.description || '',
          parent_id: category.parent_id || '',
          status: category.status,
          allow_custom_dimensions: category.allow_custom_dimensions,
        });
      } else {
        reset({
          name: '',
          slug: '',
          description: '',
          parent_id: category?.parent_id || '',
          status: 'ACTIVE',
          allow_custom_dimensions: false,
        });
      }
    }
  }, [isOpen, category, reset]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateCategoryRequest) => categoriesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categories-tree'] });
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
      queryClient.invalidateQueries({ queryKey: ['categories-tree'] });
      toast.success('Category updated successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: CategoryFormData) => {
    const requestData: CreateCategoryRequest = {
      name: data.name,
      slug: data.slug,
      description: data.description,
      parent_id: data.parent_id || undefined,
      status: data.status,
      allow_custom_dimensions: data.allow_custom_dimensions,
    };

    if (isEditing && category?.id) {
      updateMutation.mutate({ id: category.id, data: requestData });
    } else {
      createMutation.mutate(requestData);
    }
  };

  // Flatten categories for parent select
  const flattenCategories = (cats: Category[], depth = 0): { value: string; label: string }[] => {
    const result: { value: string; label: string }[] = [];
    cats.forEach((cat) => {
      // Don't allow selecting itself or its children as parent
      if (category?.id && (cat.id === category.id || cat.path?.includes(category.id))) {
        return;
      }
      result.push({
        value: cat.id,
        label: `${'—'.repeat(depth)} ${cat.name}`,
      });
      if (cat.children && cat.children.length > 0) {
        result.push(...flattenCategories(cat.children, depth + 1));
      }
    });
    return result;
  };

  const parentOptions = [
    { value: '', label: 'No parent (Root category)' },
    ...flattenCategories(categories),
  ];

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Category' : 'Create Category'}
      size="md"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <Input
          label="Name"
          placeholder="e.g., Bedsheets"
          error={errors.name?.message}
          required
          {...register('name')}
        />

        <Input
          label="Slug"
          placeholder="e.g., bedsheets"
          hint="Used in URLs. Lowercase letters, numbers, and hyphens only."
          error={errors.slug?.message}
          required
          {...register('slug')}
        />

        <div>
          <label className="label">Description</label>
          <textarea
            className="input min-h-[80px]"
            placeholder="Optional description for this category"
            {...register('description')}
          />
        </div>

        <Select label="Parent Category" options={parentOptions} {...register('parent_id')} />

        <Select
          label="Status"
          options={[
            { value: 'ACTIVE', label: 'Active' },
            { value: 'INACTIVE', label: 'Inactive' },
          ]}
          required
          {...register('status')}
        />

        <div className="flex items-center gap-3">
          <input
            type="checkbox"
            id="allow_custom_dimensions"
            className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
            {...register('allow_custom_dimensions')}
          />
          <label htmlFor="allow_custom_dimensions" className="text-sm text-gray-700">
            Allow custom dimensions for products in this category
          </label>
        </div>

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
