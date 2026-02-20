'use client';

import { useState } from 'react';

import Button from '@/components/common/Button';
import AddressForm from '@/components/checkout/AddressForm';
import api from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { Address } from '@/types';

export default function AddressesPage() {
  const { customer, checkAuth } = useAuthStore();

  const [showForm, setShowForm] = useState(false);
  const [editingAddress, setEditingAddress] = useState<Address | null>(null);
  const [saving, setSaving] = useState(false);
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const addresses = customer?.addresses || [];

  const handleAdd = async (data: Omit<Address, 'id'>) => {
    setSaving(true);
    setError(null);
    try {
      await api.post('/api/v1/store/me/addresses', data);
      await checkAuth();
      setShowForm(false);
    } catch {
      setError('Failed to add address. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleUpdate = async (data: Omit<Address, 'id'>) => {
    if (!editingAddress) return;
    setSaving(true);
    setError(null);
    try {
      await api.put(`/api/v1/store/me/addresses/${editingAddress.id}`, data);
      await checkAuth();
      setEditingAddress(null);
    } catch {
      setError('Failed to update address. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (addressId: string) => {
    const confirmed = window.confirm(
      'Are you sure you want to delete this address?',
    );
    if (!confirmed) return;

    setDeletingId(addressId);
    setError(null);
    try {
      await api.delete(`/api/v1/store/me/addresses/${addressId}`);
      await checkAuth();
    } catch {
      setError('Failed to delete address. Please try again.');
    } finally {
      setDeletingId(null);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">
          Saved Addresses
        </h2>
        {!showForm && !editingAddress && (
          <Button size="sm" onClick={() => setShowForm(true)}>
            Add Address
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Add form */}
      {showForm && (
        <div className="rounded-lg border border-border bg-white p-6">
          <h3 className="mb-4 font-semibold text-foreground">Add New Address</h3>
          <AddressForm
            onSubmit={handleAdd}
            onCancel={() => setShowForm(false)}
            loading={saving}
          />
        </div>
      )}

      {/* Edit form */}
      {editingAddress && (
        <div className="rounded-lg border border-border bg-white p-6">
          <h3 className="mb-4 font-semibold text-foreground">Edit Address</h3>
          <AddressForm
            initialData={editingAddress}
            onSubmit={handleUpdate}
            onCancel={() => setEditingAddress(null)}
            loading={saving}
          />
        </div>
      )}

      {/* Address cards */}
      {addresses.length === 0 && !showForm && (
        <div className="rounded-lg border border-border bg-white p-12 text-center">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1}
            stroke="currentColor"
            className="mx-auto mb-4 h-16 w-16 text-muted"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M15 10.5a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
            />
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1 1 15 0Z"
            />
          </svg>
          <h3 className="mb-2 text-lg font-semibold text-foreground">
            No saved addresses
          </h3>
          <p className="mb-4 text-muted">
            Add an address to speed up your checkout.
          </p>
          <Button onClick={() => setShowForm(true)}>Add Address</Button>
        </div>
      )}

      {addresses.length > 0 && !showForm && !editingAddress && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {addresses.map((addr) => (
            <div
              key={addr.id}
              className="rounded-lg border border-border bg-white p-5"
            >
              <div className="mb-3 flex items-start justify-between">
                <div>
                  <p className="font-medium text-foreground">
                    {addr.first_name} {addr.last_name}
                    {addr.is_default && (
                      <span className="ml-2 rounded bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                        Default
                      </span>
                    )}
                  </p>
                </div>
              </div>

              <div className="mb-4 text-sm text-muted">
                <p>{addr.address_line1}</p>
                {addr.address_line2 && <p>{addr.address_line2}</p>}
                <p>
                  {addr.city}, {addr.state} - {addr.postal_code}
                </p>
                <p>Phone: {addr.phone}</p>
              </div>

              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setEditingAddress(addr)}
                  className="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-background"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={() => handleDelete(addr.id)}
                  disabled={deletingId === addr.id}
                  className="rounded-md border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-50"
                >
                  {deletingId === addr.id ? 'Deleting...' : 'Delete'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
