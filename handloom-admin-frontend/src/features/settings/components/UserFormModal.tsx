import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { usersApi } from '@/features/settings/api';
import { getErrorMessage } from '@/shared/api/client';
import { Button, Input, Modal, Select } from '@/shared/components/ui';
import type { CreateUserRequest, User } from '@/features/auth/types';

const userSchema = z.object({
  first_name: z
    .string()
    .min(1, 'First name is required')
    .max(50, 'First name must be less than 50 characters'),
  last_name: z
    .string()
    .min(1, 'Last name is required')
    .max(50, 'Last name must be less than 50 characters'),
  email: z.string().email('Invalid email address'),
  phone: z.string().optional(),
  password: z.union([z.literal(''), z.string().min(8, 'Password must be at least 8 characters')]),
  role: z.enum(['ADMIN', 'OPERATOR']),
  status: z.enum(['ACTIVE', 'INACTIVE', 'PENDING']),
});

type UserFormData = z.infer<typeof userSchema>;

interface UserFormModalProps {
  isOpen: boolean;
  onClose: () => void;
  user?: User | null;
}

export function UserFormModal({ isOpen, onClose, user }: UserFormModalProps) {
  const queryClient = useQueryClient();
  const isEditing = !!user?.id;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<UserFormData>({
    resolver: zodResolver(userSchema),
    defaultValues: {
      first_name: '',
      last_name: '',
      email: '',
      phone: '',
      password: '',
      role: 'OPERATOR',
      status: 'ACTIVE',
    },
  });

  // Reset form when modal opens/closes or user changes
  useEffect(() => {
    if (isOpen) {
      if (user?.id) {
        reset({
          first_name: user.first_name,
          last_name: user.last_name,
          email: user.email,
          phone: user.phone || '',
          password: '',
          role: user.role,
          status: user.status,
        });
      } else {
        reset({
          first_name: '',
          last_name: '',
          email: '',
          phone: '',
          password: '',
          role: 'OPERATOR',
          status: 'ACTIVE',
        });
      }
    }
  }, [isOpen, user, reset]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: CreateUserRequest) => usersApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast.success('User created successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateUserRequest> }) =>
      usersApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast.success('User updated successfully');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: UserFormData) => {
    if (isEditing && user?.id) {
      const updateData: Partial<CreateUserRequest> = {
        first_name: data.first_name,
        last_name: data.last_name,
        phone: data.phone || undefined,
        role: data.role,
        status: data.status,
      };
      if (data.password) {
        updateData.password = data.password;
      }
      updateMutation.mutate({ id: user.id, data: updateData });
    } else {
      if (!data.password) {
        toast.error('Password is required for new users');
        return;
      }
      createMutation.mutate({
        first_name: data.first_name,
        last_name: data.last_name,
        email: data.email,
        phone: data.phone || undefined,
        role: data.role,
        password: data.password,
        status: data.status,
      });
    }
  };

  const isLoading = createMutation.isPending || updateMutation.isPending;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit User' : 'Create User'}
      size="md"
    >
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
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
        </div>

        <Input
          label="Email"
          type="email"
          placeholder="e.g., john@example.com"
          error={errors.email?.message}
          required
          disabled={isEditing}
          {...register('email')}
        />

        <Input
          label="Phone"
          placeholder="e.g., 9876543210"
          error={errors.phone?.message}
          {...register('phone')}
        />

        <Input
          label={isEditing ? 'New Password (leave blank to keep current)' : 'Password'}
          type="password"
          placeholder={isEditing ? 'Leave blank to keep current password' : 'Enter password'}
          error={errors.password?.message}
          required={!isEditing}
          {...register('password')}
        />

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Select
            label="Role"
            options={[
              { value: 'ADMIN', label: 'Admin' },
              { value: 'OPERATOR', label: 'Operator' },
            ]}
            error={errors.role?.message}
            required
            {...register('role')}
          />

          <Select
            label="Status"
            options={[
              { value: 'ACTIVE', label: 'Active' },
              { value: 'INACTIVE', label: 'Inactive' },
              { value: 'PENDING', label: 'Pending' },
            ]}
            required
            {...register('status')}
          />
        </div>

        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Update User' : 'Create User'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
