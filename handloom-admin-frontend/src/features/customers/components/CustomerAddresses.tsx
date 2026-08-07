import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Trash2 } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { customersApi } from '@/features/customers/api';
import { getErrorMessage } from '@/shared/api/client';
import { Button, Input } from '@/shared/components/ui';
import type { Address } from '@/shared/types/common';

import { EMPTY_ADDRESS, validateAddress } from '../lib/addressValidation';
import { addressFullName } from '../lib/displayName';
import type { Customer } from '../types';

interface CustomerAddressesProps {
  customer: Customer;
  /** Receives the updated customer each endpoint returns. */
  onChange: (customer: Customer) => void;
}

export function CustomerAddresses({ customer, onChange }: CustomerAddressesProps) {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<Address | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const addresses = customer.addresses ?? [];

  const done = (updated: Customer, message: string) => {
    onChange(updated);
    queryClient.invalidateQueries({ queryKey: ['customers'] });
    toast.success(message);
    setDraft(null);
    setEditingId(null);
    setError(null);
  };

  const saveMutation = useMutation({
    mutationFn: ({ address, addressId }: { address: Address; addressId: string | null }) =>
      addressId
        ? customersApi.updateAddress(customer.id, addressId, address)
        : customersApi.addAddress(customer.id, address),
    onSuccess: (updated, { addressId }) =>
      done(updated, addressId ? 'Address updated' : 'Address added'),
    onError: (err) => setError(getErrorMessage(err)),
  });

  const removeMutation = useMutation({
    mutationFn: (addressId: string) => customersApi.removeAddress(customer.id, addressId),
    onSuccess: (updated) => done(updated, 'Address removed'),
    onError: (err) => toast.error(getErrorMessage(err)),
  });

  const handleSave = () => {
    if (!draft) return;
    const message = validateAddress(draft);
    if (message) {
      setError(message);
      return;
    }
    saveMutation.mutate({ address: draft, addressId: editingId });
  };

  const startAdd = () => {
    setEditingId(null);
    setDraft({ ...EMPTY_ADDRESS });
    setError(null);
  };

  const startEdit = (address: Address) => {
    setEditingId(address.id ?? null);
    setDraft({ ...address });
    setError(null);
  };

  const field = (key: keyof Address, label: string, placeholder?: string) => (
    <Input
      label={label}
      placeholder={placeholder}
      value={String(draft?.[key] ?? '')}
      onChange={(e) => setDraft((d) => (d ? { ...d, [key]: e.target.value } : d))}
    />
  );

  return (
    <div className="pt-4 border-t">
      <div className="flex items-center justify-between mb-2">
        <p className="text-sm font-medium text-gray-700">Addresses</p>
        {!draft && (
          <Button
            size="sm"
            variant="secondary"
            leftIcon={<Plus className="w-3 h-3" />}
            onClick={startAdd}
          >
            Add Address
          </Button>
        )}
      </div>

      {!draft && addresses.length === 0 && (
        <p className="text-sm text-gray-500">No addresses saved.</p>
      )}

      {!draft &&
        addresses.map((address, idx) => (
          <div
            key={address.id ?? idx}
            className="text-sm text-gray-600 p-3 bg-gray-50 rounded-lg mb-2 flex justify-between gap-3"
          >
            <div>
              <p className="font-medium text-gray-900">
                {addressFullName(address) || '—'}
                {address.is_default && (
                  <span className="ml-2 text-xs text-gray-500">(default)</span>
                )}
              </p>
              <p>{address.address_line1}</p>
              {address.address_line2 && <p>{address.address_line2}</p>}
              <p>
                {address.city}, {address.state} {address.postal_code}
              </p>
              <p>{address.country}</p>
              {address.phone && <p>Phone: {address.phone}</p>}
            </div>
            <div className="flex items-start gap-1">
              {/* Addresses saved before this screen existed have no id, and the
                  endpoints key off one — those stay read-only. */}
              {address.id && (
                <>
                  <Button variant="ghost" size="sm" onClick={() => startEdit(address)}>
                    Edit
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-red-600 hover:text-red-700 hover:bg-red-50"
                    loading={removeMutation.isPending && removeMutation.variables === address.id}
                    onClick={() => removeMutation.mutate(address.id as string)}
                  >
                    <Trash2 className="w-4 h-4" />
                  </Button>
                </>
              )}
            </div>
          </div>
        ))}

      {draft && (
        <div className="space-y-3 p-3 bg-gray-50 rounded-lg">
          {error && <p className="text-sm text-red-600">{error}</p>}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {field('first_name', 'First Name', 'e.g., John')}
            {field('last_name', 'Last Name', 'e.g., Doe')}
          </div>
          {field('address_line1', 'Address Line 1', 'e.g., 123 Main Street')}
          {field('address_line2', 'Address Line 2 (Optional)', 'e.g., Apt 4B')}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            {field('city', 'City', 'e.g., Mumbai')}
            {field('state', 'State', 'e.g., Maharashtra')}
            {field('postal_code', 'PIN Code', 'e.g., 400001')}
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {field('country', 'Country')}
            {field('phone', 'Phone', 'e.g., 9876543210')}
          </div>
          <label className="flex items-center gap-2 text-sm text-gray-700">
            <input
              type="checkbox"
              checked={!!draft.is_default}
              onChange={(e) => setDraft((d) => (d ? { ...d, is_default: e.target.checked } : d))}
            />
            Set as default address
          </label>
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={() => {
                setDraft(null);
                setEditingId(null);
                setError(null);
              }}
            >
              Cancel
            </Button>
            <Button size="sm" loading={saveMutation.isPending} onClick={handleSave}>
              {editingId ? 'Update Address' : 'Save Address'}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
