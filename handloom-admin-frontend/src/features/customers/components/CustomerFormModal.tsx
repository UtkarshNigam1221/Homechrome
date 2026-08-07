import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { customersApi } from '@/features/customers/api';
import { Button, Input, Modal, Select } from '@/shared/components/ui';
import { useFormMutation } from '@/shared/hooks';

import type { CreateCustomerRequest, Customer, UpdateCustomerRequest } from '../types';

const customerSchema = z.object({
  first_name: z.string().min(1, 'First name is required').max(100),
  last_name: z.string().min(1, 'Last name is required').max(100),
  email: z.string().email('Invalid email address'),
  phone: z.string().optional(),
  status: z.enum(['ACTIVE', 'INACTIVE', 'BLOCKED']),
  // Address fields
  address_first_name: z.string().optional(),
  address_last_name: z.string().optional(),
  address_line1: z.string().optional(),
  address_line2: z.string().optional(),
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
  const isEditing = !!customer?.id;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CustomerFormData>({
    resolver: zodResolver(customerSchema),
    defaultValues: {
      first_name: '',
      last_name: '',
      email: '',
      phone: '',
      status: 'ACTIVE',
      address_first_name: '',
      address_last_name: '',
      address_line1: '',
      address_line2: '',
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
          first_name: customer.first_name || '',
          last_name: customer.last_name || '',
          email: customer.email,
          phone: customer.phone || '',
          status: customer.status,
          address_first_name: defaultAddress?.first_name || '',
          address_last_name: defaultAddress?.last_name || '',
          address_line1: defaultAddress?.address_line1 || '',
          address_line2: defaultAddress?.address_line2 || '',
          address_city: defaultAddress?.city || '',
          address_state: defaultAddress?.state || '',
          address_postal_code: defaultAddress?.postal_code || '',
          address_country: defaultAddress?.country || 'India',
          address_phone: defaultAddress?.phone || '',
        });
      } else {
        reset({
          first_name: '',
          last_name: '',
          email: '',
          phone: '',
          status: 'ACTIVE',
          address_first_name: '',
          address_last_name: '',
          address_line1: '',
          address_line2: '',
          address_city: '',
          address_state: '',
          address_postal_code: '',
          address_country: 'India',
          address_phone: '',
        });
      }
    }
  }, [isOpen, customer, reset]);

  const { isLoading, onSubmit: submitMutation } = useFormMutation<
    CreateCustomerRequest,
    UpdateCustomerRequest
  >({
    queryKey: 'customers',
    createFn: customersApi.create,
    updateFn: customersApi.update,
    entityName: 'Customer',
    onSuccess: onClose,
  });

  const onSubmit = (data: CustomerFormData) => {
    const hasAddress = data.address_line1 || data.address_city;

    const requestData: CreateCustomerRequest = {
      first_name: data.first_name,
      last_name: data.last_name,
      email: data.email,
      phone: data.phone || undefined,
      address: hasAddress
        ? {
            first_name: data.address_first_name || data.first_name,
            last_name: data.address_last_name || data.last_name,
            address_line1: data.address_line1 || '',
            address_line2: data.address_line2 || undefined,
            city: data.address_city || '',
            state: data.address_state || '',
            postal_code: data.address_postal_code || '',
            country: data.address_country || 'India',
            phone: data.address_phone || data.phone,
            is_default: true,
          }
        : undefined,
    };

    // UpdateCustomerRequest carries no address — the backend manages those
    // through /customers/{id}/addresses, so edits here only touch the profile.
    const updateData: UpdateCustomerRequest = {
      first_name: data.first_name,
      last_name: data.last_name,
      phone: data.phone || undefined,
      status: data.status,
    };

    submitMutation(customer?.id, requestData, updateData);
  };

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
              label="First Name"
              placeholder="e.g., John"
              error={errors.first_name?.message}
              required
              {...register('first_name')}
            />

            <Input
              label="Last Name"
              placeholder="e.g., Doe"
              error={errors.last_name?.message}
              required
              {...register('last_name')}
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
                { value: 'BLOCKED', label: 'Blocked' },
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
              label="Contact First Name"
              placeholder="e.g., John"
              {...register('address_first_name')}
            />

            <Input
              label="Contact Last Name"
              placeholder="e.g., Doe"
              {...register('address_last_name')}
            />

            <Input label="Phone" placeholder="e.g., 9876543210" {...register('address_phone')} />

            <div className="md:col-span-2">
              <Input
                label="Address Line 1"
                placeholder="e.g., 123 Main Street"
                {...register('address_line1')}
              />
            </div>

            <div className="md:col-span-2">
              <Input
                label="Address Line 2 (Optional)"
                placeholder="e.g., Apt 4B, near the park"
                {...register('address_line2')}
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
