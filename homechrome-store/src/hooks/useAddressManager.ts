'use client';

import { useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { useAuthStore } from '@/stores/auth';
import { Address } from '@/types';

export function useAddressManager() {
  const customer = useAuthStore((s) => s.customer);
  const checkAuth = useAuthStore((s) => s.checkAuth);

  const [showForm, setShowForm] = useState(false);
  const [editingAddress, setEditingAddress] = useState<Address | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const addresses = customer?.addresses || [];

  const add = async (data: Omit<Address, 'id'>) => {
    setSaving(true);
    setError(null);
    try {
      await api.post(ROUTES.ME.ADDRESSES, data);
      await checkAuth();
      setShowForm(false);
    } catch {
      setError('Failed to add address. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const update = async (data: Omit<Address, 'id'>) => {
    if (!editingAddress) return;
    setSaving(true);
    setError(null);
    try {
      await api.put(ROUTES.ME.ADDRESS(editingAddress.id), data);
      await checkAuth();
      setEditingAddress(null);
    } catch {
      setError('Failed to update address. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const remove = async (addressId: string) => {
    setDeletingId(addressId);
    setError(null);
    try {
      await api.delete(ROUTES.ME.ADDRESS(addressId));
      await checkAuth();
    } catch {
      setError('Failed to delete address. Please try again.');
    } finally {
      setDeletingId(null);
    }
  };

  return {
    addresses,
    showForm,
    editingAddress,
    saving,
    deletingId,
    error,
    setShowForm,
    setEditingAddress,
    add,
    update,
    remove,
  };
}
