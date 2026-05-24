'use client';

import { Button, Checkbox, Group, SimpleGrid, Stack, TextInput } from '@mantine/core';
import { useForm } from '@mantine/form';

import { Address } from '@/types';

interface AddressFormProps {
  initialData?: Address | null;
  onSubmit: (data: Omit<Address, 'id'>) => Promise<void>;
  onCancel: () => void;
  loading?: boolean;
}

export default function AddressForm({
  initialData,
  onSubmit,
  onCancel,
  loading = false,
}: AddressFormProps) {
  const form = useForm({
    initialValues: {
      first_name: initialData?.first_name || '',
      last_name: initialData?.last_name || '',
      phone: initialData?.phone || '',
      address_line1: initialData?.address_line1 || '',
      address_line2: initialData?.address_line2 || '',
      city: initialData?.city || '',
      state: initialData?.state || '',
      postal_code: initialData?.postal_code || '',
      country: initialData?.country || 'India',
      is_default: initialData?.is_default || false,
    },
    validate: {
      first_name: (v) => (!v.trim() ? 'First name is required' : null),
      last_name: (v) => (!v.trim() ? 'Last name is required' : null),
      phone: (v) => {
        if (!v.trim()) return 'Phone is required';
        if (!/^[6-9]\d{9}$/.test(v.trim())) return 'Enter a valid 10-digit mobile number';
        return null;
      },
      address_line1: (v) => (!v.trim() ? 'Address is required' : null),
      city: (v) => (!v.trim() ? 'City is required' : null),
      state: (v) => (!v.trim() ? 'State is required' : null),
      postal_code: (v) => {
        if (!v.trim()) return 'PIN code is required';
        if (!/^\d{6}$/.test(v.trim())) return 'Enter a valid 6-digit PIN code';
        return null;
      },
    },
  });

  const handleSubmit = form.onSubmit(async (values) => {
    await onSubmit({
      first_name: values.first_name.trim(),
      last_name: values.last_name.trim(),
      phone: values.phone.trim(),
      address_line1: values.address_line1.trim(),
      address_line2: values.address_line2.trim() || undefined,
      city: values.city.trim(),
      state: values.state.trim(),
      postal_code: values.postal_code.trim(),
      country: values.country.trim(),
      is_default: values.is_default,
    });
  });

  return (
    <form onSubmit={handleSubmit}>
      <Stack gap="md">
        <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
          <TextInput
            label="First Name"
            placeholder="First name"
            {...form.getInputProps('first_name')}
          />
          <TextInput
            label="Last Name"
            placeholder="Last name"
            {...form.getInputProps('last_name')}
          />
        </SimpleGrid>

        <TextInput
          label="Phone Number"
          type="tel"
          placeholder="10-digit mobile number"
          maxLength={10}
          {...form.getInputProps('phone')}
        />

        <TextInput
          label="Address Line 1"
          placeholder="House number, street name"
          {...form.getInputProps('address_line1')}
        />

        <TextInput
          label="Address Line 2 (Optional)"
          placeholder="Apartment, landmark, etc."
          {...form.getInputProps('address_line2')}
        />

        <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
          <TextInput
            label="City"
            placeholder="City"
            {...form.getInputProps('city')}
          />
          <TextInput
            label="State"
            placeholder="State"
            {...form.getInputProps('state')}
          />
          <TextInput
            label="PIN Code"
            placeholder="6-digit PIN"
            maxLength={6}
            {...form.getInputProps('postal_code')}
          />
        </SimpleGrid>

        <Checkbox
          label="Set as default address"
          {...form.getInputProps('is_default', { type: 'checkbox' })}
        />

        <Group gap="sm" mt="xs">
          <Button type="submit" color="brand" loading={loading}>
            {initialData ? 'Update Address' : 'Save Address'}
          </Button>
          <Button type="button" variant="outline" color="navy" onClick={onCancel}>
            Cancel
          </Button>
        </Group>
      </Stack>
    </form>
  );
}
