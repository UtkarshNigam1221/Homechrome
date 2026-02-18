import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { artisansApi } from '@/features/artisans/api';
import { getErrorMessage } from '@/shared/api/client';
import { Button, Input, Modal, Select } from '@/shared/components/ui';
import type { Artisan, CreateArtisanRequest, UpdateArtisanRequest } from '../types';

const artisanSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be less than 100 characters'),
  email: z.string().email('Invalid email').optional().or(z.literal('')),
  phone: z.string().min(10, 'Phone must be at least 10 digits'),
  craft_type: z.string().optional(),
  skills: z.string().optional(),
  city: z.string().min(1, 'City is required'),
  state: z.string().min(1, 'State is required'),
  country: z.string().min(1, 'Country is required'),
  bio: z.string().optional(),
  account_name: z.string().optional(),
  account_number: z.string().optional(),
  bank_name: z.string().optional(),
  ifsc_code: z.string().optional(),
  upi_id: z.string().optional(),
  status: z.enum(['ACTIVE', 'INACTIVE', 'SUSPENDED']),
});

type ArtisanFormData = z.infer<typeof artisanSchema>;

interface ArtisanFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  artisan?: Artisan | null;
}

export function ArtisanFormModal({ isOpen, onClose, artisan }: ArtisanFormModalProps) {
  const queryClient = useQueryClient();
  const isEditing = !!artisan?.id;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<ArtisanFormData>({
    resolver: zodResolver(artisanSchema),
    defaultValues: {
      name: '',
      email: '',
      phone: '',
      craft_type: '',
      skills: '',
      city: '',
      state: '',
      country: 'India',
      bio: '',
      account_name: '',
      account_number: '',
      bank_name: '',
      ifsc_code: '',
      upi_id: '',
      status: 'ACTIVE',
    },
  });

  // Reset form when modal opens/closes or artisan changes
  useEffect(() => {
    if (isOpen) {
      if (artisan?.id) {
        reset({
          name: artisan.name,
          email: artisan.email || '',
          phone: artisan.phone,
          craft_type: artisan.craft_type || '',
          skills: artisan.skills?.join(', ') || '',
          city: artisan.location?.city || '',
          state: artisan.location?.state || '',
          country: artisan.location?.country || 'India',
          bio: artisan.bio || '',
          account_name: artisan.bank_details?.account_name || '',
          account_number: artisan.bank_details?.account_number || '',
          bank_name: artisan.bank_details?.bank_name || '',
          ifsc_code: artisan.bank_details?.ifsc_code || '',
          upi_id: artisan.bank_details?.upi_id || '',
          status: artisan.status,
        });
      } else {
        reset({
          name: '',
          email: '',
          phone: '',
          craft_type: '',
          skills: '',
          city: '',
          state: '',
          country: 'India',
          bio: '',
          account_name: '',
          account_number: '',
          bank_name: '',
          ifsc_code: '',
          upi_id: '',
          status: 'ACTIVE',
        });
      }
    }
  }, [isOpen, artisan, reset]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateArtisanRequest) => artisansApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['artisans'] });
      toast.success('Artisan created successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateArtisanRequest> }) =>
      artisansApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['artisans'] });
      toast.success('Artisan updated successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: ArtisanFormData) => {
    const hasBankDetails =
      data.account_name || data.account_number || data.bank_name || data.ifsc_code;

    const requestData: CreateArtisanRequest = {
      name: data.name,
      email: data.email || undefined,
      phone: data.phone,
      craft_type: data.craft_type || undefined,
      skills: data.skills
        ? data.skills
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean)
        : undefined,
      location: {
        city: data.city,
        state: data.state,
        country: data.country,
      },
      bio: data.bio || undefined,
      bank_details: hasBankDetails
        ? {
            account_name: data.account_name || '',
            account_number: data.account_number || '',
            bank_name: data.bank_name || '',
            ifsc_code: data.ifsc_code || '',
            upi_id: data.upi_id || undefined,
          }
        : undefined,
    };

    if (isEditing && artisan?.id) {
      updateMutation.mutate({
        id: artisan.id,
        data: { ...requestData, status: data.status } as UpdateArtisanRequest,
      });
    } else {
      createMutation.mutate(requestData);
    }
  };

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Artisan' : 'Add Artisan'}
      size="lg"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Basic Information */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Basic Information</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              label="Full Name"
              placeholder="e.g., Ramesh Kumar"
              error={errors.name?.message}
              required
              {...register('name')}
            />

            <Input
              label="Phone"
              placeholder="e.g., 9876543210"
              error={errors.phone?.message}
              required
              {...register('phone')}
            />

            <Input
              label="Email"
              type="email"
              placeholder="e.g., artisan@email.com"
              error={errors.email?.message}
              {...register('email')}
            />

            <Input
              label="Craft Type"
              placeholder="e.g., Weaving, Embroidery"
              {...register('craft_type')}
            />

            <div className="md:col-span-2">
              <Input
                label="Skills"
                placeholder="e.g., Silk weaving, Block printing, Zari work"
                hint="Comma-separated list of skills"
                {...register('skills')}
              />
            </div>

            <div className="md:col-span-2">
              <label className="label">Bio</label>
              <textarea
                className="input min-h-[80px]"
                placeholder="Brief description about the artisan..."
                {...register('bio')}
              />
            </div>
          </div>
        </div>

        {/* Location */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Location</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Input
              label="City"
              placeholder="e.g., Varanasi"
              error={errors.city?.message}
              required
              {...register('city')}
            />

            <Input
              label="State"
              placeholder="e.g., Uttar Pradesh"
              error={errors.state?.message}
              required
              {...register('state')}
            />

            <Input
              label="Country"
              placeholder="e.g., India"
              error={errors.country?.message}
              required
              {...register('country')}
            />
          </div>
        </div>

        {/* Bank Details */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Bank Details (Optional)</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              label="Account Holder Name"
              placeholder="e.g., Ramesh Kumar"
              {...register('account_name')}
            />

            <Input
              label="Account Number"
              placeholder="e.g., 1234567890"
              {...register('account_number')}
            />

            <Input
              label="Bank Name"
              placeholder="e.g., State Bank of India"
              {...register('bank_name')}
            />

            <Input label="IFSC Code" placeholder="e.g., SBIN0001234" {...register('ifsc_code')} />

            <Input label="UPI ID" placeholder="e.g., artisan@upi" {...register('upi_id')} />

            <Select
              label="Status"
              options={[
                { value: 'ACTIVE', label: 'Active' },
                { value: 'INACTIVE', label: 'Inactive' },
                { value: 'SUSPENDED', label: 'Suspended' },
              ]}
              required
              {...register('status')}
            />
          </div>
        </div>

        {/* Form Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Update Artisan' : 'Add Artisan'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
