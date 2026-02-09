import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { Controller, useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { categoriesApi, designsApi, getErrorMessage } from '../../api';
import { Button, ImageUpload, Input, Modal, Select } from '../../components/common';
import type { Category, CreateDesignRequest, Design, UpdateDesignRequest } from '../../types';

const designSchema = z.object({
  name: z.string().min(1, 'Name is required').max(200, 'Name must be less than 200 characters'),
  slug: z
    .string()
    .min(1, 'Slug is required')
    .regex(/^[a-z0-9-]+$/, 'Slug must be lowercase letters, numbers, and hyphens only'),
  category_id: z.string().min(1, 'Category is required'),
  description: z.string().optional(),
  preview_image: z.string().optional(),
  status: z.enum(['ACTIVE', 'INACTIVE']),
});

type DesignFormData = z.infer<typeof designSchema>;

interface DesignFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  design?: Design | null;
}

export function DesignFormModal({ isOpen, onClose, design }: DesignFormModalProps) {
  const queryClient = useQueryClient();
  const isEditing = !!design?.id;

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
    setValue,
    control,
    formState: { errors },
  } = useForm<DesignFormData>({
    resolver: zodResolver(designSchema),
    defaultValues: {
      name: '',
      slug: '',
      category_id: '',
      description: '',
      preview_image: '',
      status: 'ACTIVE',
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

  // Reset form when modal opens/closes or design changes
  useEffect(() => {
    if (isOpen) {
      if (design?.id) {
        // Extract image URL from either preview_image_url or images array
        const imageUrl = design.preview_image_url || design.images?.[0]?.url || '';
        reset({
          name: design.name,
          slug: design.slug,
          category_id: design.category_id,
          description: design.description || '',
          preview_image: imageUrl,
          status: design.status,
        });
      } else {
        reset({
          name: '',
          slug: '',
          category_id: '',
          description: '',
          preview_image: '',
          status: 'ACTIVE',
        });
      }
    }
  }, [isOpen, design, reset]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateDesignRequest) => designsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['designs'] });
      toast.success('Design created successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateDesignRequest }) =>
      designsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['designs'] });
      toast.success('Design updated successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: DesignFormData) => {
    const requestData: CreateDesignRequest = {
      name: data.name,
      slug: data.slug,
      category_id: data.category_id,
      description: data.description || undefined,
      images: data.preview_image
        ? [
            {
              url: data.preview_image,
              alt_text: data.name,
              is_primary: true,
              sort_order: 0,
            },
          ]
        : undefined,
    };

    if (isEditing && design?.id) {
      const updateData: UpdateDesignRequest = { ...requestData, status: data.status };
      updateMutation.mutate({
        id: design.id,
        data: updateData,
      });
    } else {
      createMutation.mutate(requestData);
    }
  };

  // Map categories to select options (flat list, no hierarchy)
  const flattenCategories = (cats: Category[]): { value: string; label: string }[] => {
    return cats.map((cat) => ({
      value: cat.id,
      label: cat.name,
    }));
  };

  // Handle various response formats from the API
  // Backend returns { categories: [...], pagination: {...} }
  const extractItems = (data: unknown, key?: string): Category[] => {
    if (!data) return [];
    if (Array.isArray(data)) return data as Category[];
    if (typeof data === 'object' && data !== null) {
      const record = data as Record<string, unknown>;
      if (key && key in record) return record[key] as Category[];
      if ('items' in record) return record.items as Category[];
      if ('data' in record) return Array.isArray(record.data) ? (record.data as Category[]) : [];
    }
    return [];
  };

  const categories = extractItems(categoriesData, 'categories');
  const categoryOptions = flattenCategories(categories);

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Design' : 'Create Design'}
      size="md"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <Input
          label="Design Name"
          placeholder="e.g., Paisley Pattern"
          error={errors.name?.message}
          required
          {...register('name')}
        />

        <Input
          label="Slug"
          placeholder="e.g., paisley-pattern"
          hint="Used in URLs. Lowercase letters, numbers, and hyphens only."
          error={errors.slug?.message}
          required
          {...register('slug')}
        />

        <Select
          label="Category"
          options={categoryOptions}
          placeholder="Select a category"
          error={errors.category_id?.message}
          required
          {...register('category_id')}
        />

        <div>
          <label className="label">Description</label>
          <textarea
            className="input min-h-[80px]"
            placeholder="Describe this design..."
            {...register('description')}
          />
        </div>

        <Controller
          name="preview_image"
          control={control}
          render={({ field }) => (
            <ImageUpload
              label="Preview Image"
              value={field.value}
              onChange={(value) => field.onChange(Array.isArray(value) ? value[0] : value)}
              hint="Upload a preview image for this design"
              error={errors.preview_image?.message}
            />
          )}
        />

        <Select
          label="Status"
          options={[
            { value: 'ACTIVE', label: 'Active' },
            { value: 'INACTIVE', label: 'Inactive' },
          ]}
          required
          {...register('status')}
        />

        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Update Design' : 'Create Design'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
