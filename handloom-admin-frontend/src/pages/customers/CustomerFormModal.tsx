import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { customersApi, getErrorMessage } from '@/api';
import { Button, Input, Modal, Select } from '@/components/common';
import type { CreateCustomerRequest, Customer, UpdateCustomerRequest } from '@/types';

const customerSchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be less than 100 characters'),
  email: z.string().email('Invalid email address'),
  phone: z.string().optional(),
  status: z.enum(['ACTIVE', 'INACTIVE', 'SUSPENDED']),
  // Address fields
  address_name: z.string().optional(),
  address_street: z.string().optional(),
  address_city: z.string().optional(),
  address_state: z.string().optional(),
  address_postal_code: z.string().optional(),
  address_country: z.string().optional(),
  address_phone: z.string().optional(),
});

type CustomerFormData = z.infer<typeof customerSchema>;

interface CustomerFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  customer?: Customer | null;
}

export function CustomerFormModal({ isOpen, onClose, customer }: CustomerFormModalProps) {
  const queryClient = useQueryClient();
  const isEditing = !!customer?.id;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CustomerFormData>({
    resolver: zodResolver(customerSchema),
    defaultValues: {
      name: '',
      email: '',
      phone: '',
      status: 'ACTIVE',
      address_name: '',
      address_street: '',
      address_city: '',
      address_state: '',
      address_postal_code: '',
      address_country: 'India',
      address_phone: '',
    },
  });

  // Reset form when modal opens/closes or customer changes
  useEffect(() => {
    if (isOpen) {
      const defaultAddress = customer?.addresses?.[0];
      if (customer?.id) {
        reset({
          name: customer.name || `${customer.first_name || ''} ${customer.last_name || ''}`.trim(),
          email: customer.email,
          phone: customer.phone || '',
          status: customer.status,
          address_name: defaultAddress?.name || '',
          address_street: defaultAddress?.street || '',
          address_city: defaultAddress?.city || '',
          address_state: defaultAddress?.state || '',
          address_postal_code: defaultAddress?.postal_code || '',
          address_country: defaultAddress?.country || 'India',
          address_phone: defaultAddress?.phone || '',
        });
      } else {
        reset({
          name: '',
          email: '',
          phone: '',
          status: 'ACTIVE',
          address_name: '',
          address_street: '',
          address_city: '',
          address_state: '',
          address_postal_code: '',
          address_country: 'India',
          address_phone: '',
        });
      }
    }
  }, [isOpen, customer, reset]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateCustomerRequest) => customersApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] });
      toast.success('Customer created successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateCustomerRequest> }) =>
      customersApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['customers'] });
      toast.success('Customer updated successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: CustomerFormData) => {
    const hasAddress = data.address_street || data.address_city;

    const requestData: CreateCustomerRequest = {
      name: data.name,
      email: data.email,
      phone: data.phone || undefined,
      addresses: hasAddress
        ? [
            {
              type: 'shipping' as const,
              name: data.address_name || data.name,
              street: data.address_street || '',
              city: data.address_city || '',
              state: data.address_state || '',
              postal_code: data.address_postal_code || '',
              country: data.address_country || 'India',
              phone: data.address_phone || data.phone,
              is_default: true,
            },
          ]
        : undefined,
    };

    if (isEditing && customer?.id) {
      updateMutation.mutate({
        id: customer.id,
        data: { ...requestData, status: data.status } as UpdateCustomerRequest,
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
      title={isEditing ? 'Edit Customer' : 'Add Customer'}
      size="lg"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Basic Information */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Basic Information</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              label="Full Name"
              placeholder="e.g., John Doe"
              error={errors.name?.message}
              required
              {...register('name')}
            />

            <Input
              label="Email"
              type="email"
              placeholder="e.g., john@example.com"
              error={errors.email?.message}
              required
              {...register('email')}
            />

            <Input
              label="Phone"
              placeholder="e.g., 9876543210"
              error={errors.phone?.message}
              {...register('phone')}
            />

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

        {/* Default Address */}
        <div>
          <h3 className="text-sm font-medium text-gray-700 mb-3">Default Address (Optional)</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Input
              label="Contact Name"
              placeholder="e.g., John Doe"
              {...register('address_name')}
            />

            <Input label="Phone" placeholder="e.g., 9876543210" {...register('address_phone')} />

            <div className="md:col-span-2">
              <Input
                label="Street Address"
                placeholder="e.g., 123 Main Street, Apt 4B"
                {...register('address_street')}
              />
            </div>

            <Input label="City" placeholder="e.g., Mumbai" {...register('address_city')} />

            <Input label="State" placeholder="e.g., Maharashtra" {...register('address_state')} />

            <Input
              label="Postal Code"
              placeholder="e.g., 400001"
              {...register('address_postal_code')}
            />

            <Input label="Country" placeholder="e.g., India" {...register('address_country')} />
          </div>
        </div>

        {/* Form Actions */}
        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Update Customer' : 'Add Customer'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
