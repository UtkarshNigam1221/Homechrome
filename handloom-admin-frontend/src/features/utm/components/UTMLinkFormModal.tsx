import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { Copy } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { categoriesApi } from '@/features/categories/api';
import { productsApi } from '@/features/products/api';
import { utmLinksApi } from '@/features/utm/api';
import { Button, Input, Modal, Select } from '@/shared/components/ui';
import { useDebounce, useFormMutation } from '@/shared/hooks';

import type { CreateUTMLinkRequest, UTMLink } from '../types';

// Mirrors the backend's utmValuePattern + MaxUTMValueLen. The storefront lowercases
// and truncates on capture, so anything outside this would produce a link whose
// params never match its own analytics rows.
const utmValue = z
  .string()
  .min(1, 'Required')
  .max(32, 'Must be at most 32 characters')
  .regex(/^[a-z0-9._-]+$/, 'Lowercase letters, digits, dot, underscore or hyphen only');

const utmLinkSchema = z
  .object({
    name: z.string().min(1, 'Name is required'),
    dest_type: z.enum(['HOME', 'CATEGORY', 'PRODUCT']),
    dest_slug: z.string().optional(),
    utm_source: utmValue,
    utm_medium: utmValue,
    utm_campaign: utmValue,
  })
  .refine((d) => d.dest_type === 'HOME' || !!d.dest_slug, {
    message: 'Pick a destination page',
    path: ['dest_slug'],
  });

type UTMLinkFormData = z.infer<typeof utmLinkSchema>;

const emptyForm: UTMLinkFormData = {
  name: '',
  dest_type: 'HOME',
  dest_slug: '',
  utm_source: '',
  utm_medium: '',
  utm_campaign: '',
};

// Advisory only — the stored URL is whatever the backend returns. Kept in step with
// buildUTMURL in internal/service/utm_link_service.go.
const STORE_BASE_URL = 'https://www.homechrome.in';

function buildPreview(data: UTMLinkFormData): string {
  let path = '/';
  if (data.dest_type === 'CATEGORY') path = `/c/${data.dest_slug ?? ''}`;
  if (data.dest_type === 'PRODUCT') path = `/p/${data.dest_slug ?? ''}`;

  const params = new URLSearchParams({
    utm_source: data.utm_source,
    utm_medium: data.utm_medium,
    utm_campaign: data.utm_campaign,
  });
  params.sort(); // Go's url.Values.Encode sorts keys; match it so the preview is byte-identical

  return `${STORE_BASE_URL}${path}?${params.toString()}`;
}

interface UTMLinkFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  link?: UTMLink | null;
}

export function UTMLinkFormModal({ isOpen, onClose, link }: UTMLinkFormModalProps) {
  const isEditing = !!link?.id;
  const [productSearch, setProductSearch] = useState('');
  const debouncedProductSearch = useDebounce(productSearch, 300);

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<UTMLinkFormData>({
    resolver: zodResolver(utmLinkSchema),
    defaultValues: emptyForm,
  });

  const values = watch();
  const destType = values.dest_type;

  const { data: categories } = useQuery({
    queryKey: ['categories', { forUtm: true }],
    queryFn: () => categoriesApi.list({ limit: 100 }),
    enabled: isOpen && destType === 'CATEGORY',
  });

  const { data: products } = useQuery({
    queryKey: ['products', { forUtm: true, search: debouncedProductSearch }],
    queryFn: () => productsApi.list({ limit: 50, search: debouncedProductSearch || undefined }),
    enabled: isOpen && destType === 'PRODUCT',
  });

  useEffect(() => {
    if (!isOpen) return;
    setProductSearch('');
    reset(
      link?.id
        ? {
            name: link.name,
            dest_type: link.dest_type,
            dest_slug: link.dest_slug ?? '',
            utm_source: link.utm_source,
            utm_medium: link.utm_medium,
            utm_campaign: link.utm_campaign,
          }
        : emptyForm
    );
  }, [isOpen, link, reset]);

  const { isLoading, onSubmit: submitMutation } = useFormMutation<
    CreateUTMLinkRequest,
    Partial<CreateUTMLinkRequest>
  >({
    queryKey: 'utm-links',
    createFn: utmLinksApi.create,
    updateFn: utmLinksApi.update,
    entityName: 'UTM link',
    onSuccess: onClose,
  });

  const onSubmit = (data: UTMLinkFormData) => {
    const requestData: CreateUTMLinkRequest = {
      name: data.name,
      dest_type: data.dest_type,
      dest_slug: data.dest_type === 'HOME' ? undefined : data.dest_slug,
      utm_source: data.utm_source,
      utm_medium: data.utm_medium,
      utm_campaign: data.utm_campaign,
    };

    submitMutation(link?.id, requestData, requestData);
  };

  const preview = buildPreview(values);

  const handleCopyPreview = async () => {
    await navigator.clipboard.writeText(preview);
    toast.success('Link copied');
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit UTM Link' : 'Create UTM Link'}
      size="md"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <Input
          label="Name"
          placeholder="e.g., Diwali Google Ads — sarees"
          hint="Internal label, not part of the link"
          error={errors.name?.message}
          required
          {...register('name')}
        />

        <Select
          label="Destination"
          options={[
            { value: 'HOME', label: 'Home page' },
            { value: 'CATEGORY', label: 'Category page' },
            { value: 'PRODUCT', label: 'Product page' },
          ]}
          error={errors.dest_type?.message}
          required
          {...register('dest_type')}
        />

        {destType === 'CATEGORY' && (
          <Select
            label="Category"
            placeholder="Select a category"
            options={(categories?.items ?? []).map((c) => ({ value: c.slug, label: c.name }))}
            error={errors.dest_slug?.message}
            required
            {...register('dest_slug')}
          />
        )}

        {destType === 'PRODUCT' && (
          <div className="space-y-2">
            <Input
              label="Find product"
              placeholder="Search by name..."
              value={productSearch}
              onChange={(e) => setProductSearch(e.target.value)}
            />
            <Select
              label="Product"
              placeholder="Select a product"
              options={(products?.items ?? []).map((p) => ({ value: p.slug, label: p.name }))}
              hint="Showing up to 50 matches — narrow the search if the product is missing"
              error={errors.dest_slug?.message}
              required
              {...register('dest_slug')}
            />
          </div>
        )}

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Input
            label="utm_source"
            placeholder="google"
            error={errors.utm_source?.message}
            required
            {...register('utm_source')}
          />
          <Input
            label="utm_medium"
            placeholder="cpc"
            error={errors.utm_medium?.message}
            required
            {...register('utm_medium')}
          />
          <Input
            label="utm_campaign"
            placeholder="diwali_2026"
            error={errors.utm_campaign?.message}
            required
            {...register('utm_campaign')}
          />
        </div>

        <div className="rounded-lg bg-gray-50 border border-gray-200 p-3">
          <div className="flex items-center justify-between gap-2 mb-1">
            <span className="text-xs font-medium text-gray-500 uppercase tracking-wide">
              Preview
            </span>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleCopyPreview}
              leftIcon={<Copy className="w-3 h-3" />}
            >
              Copy
            </Button>
          </div>
          <p className="font-mono text-xs break-all text-gray-700">{preview}</p>
        </div>

        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Update Link' : 'Create Link'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
