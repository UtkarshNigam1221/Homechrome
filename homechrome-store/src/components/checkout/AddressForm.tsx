'use client';

import { useState } from 'react';

import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import FormField from '@/components/ui/form-field';
import { Label } from '@/components/ui/label';
import { Address } from '@/types';

interface AddressFormData {
  first_name: string;
  last_name: string;
  phone: string;
  address_line1: string;
  address_line2: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
  is_default: boolean;
}

const EMPTY_FORM: AddressFormData = {
  first_name: '',
  last_name: '',
  phone: '',
  address_line1: '',
  address_line2: '',
  city: '',
  state: '',
  postal_code: '',
  country: 'India',
  is_default: false,
};

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
  const [form, setForm] = useState<AddressFormData>(() =>
    initialData
      ? {
          first_name: initialData.first_name,
          last_name: initialData.last_name,
          phone: initialData.phone,
          address_line1: initialData.address_line1,
          address_line2: initialData.address_line2 || '',
          city: initialData.city,
          state: initialData.state,
          postal_code: initialData.postal_code,
          country: initialData.country || 'India',
          is_default: initialData.is_default || false,
        }
      : EMPTY_FORM,
  );
  const [errors, setErrors] = useState<Partial<Record<keyof AddressFormData, string>>>({});

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof AddressFormData, string>> = {};

    if (!form.first_name.trim()) newErrors.first_name = 'First name is required';
    if (!form.last_name.trim()) newErrors.last_name = 'Last name is required';
    if (!form.phone.trim()) {
      newErrors.phone = 'Phone is required';
    } else if (!/^[6-9]\d{9}$/.test(form.phone.trim())) {
      newErrors.phone = 'Enter a valid 10-digit mobile number';
    }
    if (!form.address_line1.trim()) newErrors.address_line1 = 'Address is required';
    if (!form.city.trim()) newErrors.city = 'City is required';
    if (!form.state.trim()) newErrors.state = 'State is required';
    if (!form.postal_code.trim()) {
      newErrors.postal_code = 'PIN code is required';
    } else if (!/^\d{6}$/.test(form.postal_code.trim())) {
      newErrors.postal_code = 'Enter a valid 6-digit PIN code';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    await onSubmit({
      first_name: form.first_name.trim(),
      last_name: form.last_name.trim(),
      phone: form.phone.trim(),
      address_line1: form.address_line1.trim(),
      address_line2: form.address_line2.trim() || undefined,
      city: form.city.trim(),
      state: form.state.trim(),
      postal_code: form.postal_code.trim(),
      country: form.country.trim(),
      is_default: form.is_default,
    });
  };

  const handleChange = (field: keyof AddressFormData, value: string | boolean) => {
    setForm((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <FormField
          label="First Name"
          value={form.first_name}
          onChange={(e) => handleChange('first_name', e.target.value)}
          error={errors.first_name}
          placeholder="First name"
        />
        <FormField
          label="Last Name"
          value={form.last_name}
          onChange={(e) => handleChange('last_name', e.target.value)}
          error={errors.last_name}
          placeholder="Last name"
        />
      </div>

      <FormField
        label="Phone Number"
        type="tel"
        value={form.phone}
        onChange={(e) => handleChange('phone', e.target.value)}
        error={errors.phone}
        placeholder="10-digit mobile number"
        maxLength={10}
      />

      <FormField
        label="Address Line 1"
        value={form.address_line1}
        onChange={(e) => handleChange('address_line1', e.target.value)}
        error={errors.address_line1}
        placeholder="House number, street name"
      />

      <FormField
        label="Address Line 2 (Optional)"
        value={form.address_line2}
        onChange={(e) => handleChange('address_line2', e.target.value)}
        placeholder="Apartment, landmark, etc."
      />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <FormField
          label="City"
          value={form.city}
          onChange={(e) => handleChange('city', e.target.value)}
          error={errors.city}
          placeholder="City"
        />
        <FormField
          label="State"
          value={form.state}
          onChange={(e) => handleChange('state', e.target.value)}
          error={errors.state}
          placeholder="State"
        />
        <FormField
          label="PIN Code"
          value={form.postal_code}
          onChange={(e) => handleChange('postal_code', e.target.value)}
          error={errors.postal_code}
          placeholder="6-digit PIN"
          maxLength={6}
        />
      </div>

      <Label className="cursor-pointer gap-2">
        <Checkbox
          checked={form.is_default}
          onCheckedChange={(checked) => handleChange('is_default', !!checked)}
        />
        Set as default address
      </Label>

      <div className="flex gap-3 pt-2">
        <Button type="submit" loading={loading}>
          {initialData ? 'Update Address' : 'Save Address'}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
