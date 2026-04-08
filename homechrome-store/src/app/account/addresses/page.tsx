'use client';

import { MapPinIcon } from '@heroicons/react/24/outline';
import { useState } from 'react';

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import Button from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { EmptyState } from '@/components/ui/empty-state';
import AddressForm from '@/components/checkout/AddressForm';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
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
      await api.post(ROUTES.ME.ADDRESSES, data);
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
      await api.put(ROUTES.ME.ADDRESS(editingAddress.id), data);
      await checkAuth();
      setEditingAddress(null);
    } catch {
      setError('Failed to update address. Please try again.');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (addressId: string) => {
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

      {showForm && (
        <Card>
          <CardHeader>
            <CardTitle>Add New Address</CardTitle>
          </CardHeader>
          <CardContent>
            <AddressForm
              onSubmit={handleAdd}
              onCancel={() => setShowForm(false)}
              loading={saving}
            />
          </CardContent>
        </Card>
      )}

      {editingAddress && (
        <Card>
          <CardHeader>
            <CardTitle>Edit Address</CardTitle>
          </CardHeader>
          <CardContent>
            <AddressForm
              initialData={editingAddress}
              onSubmit={handleUpdate}
              onCancel={() => setEditingAddress(null)}
              loading={saving}
            />
          </CardContent>
        </Card>
      )}

      {addresses.length === 0 && !showForm && (
        <Card className="p-6">
          <EmptyState
            icon={<MapPinIcon strokeWidth={1} className="h-16 w-16 text-muted-foreground" />}
            title="No saved addresses"
            description="Add an address to speed up your checkout."
          />
          <div className="text-center">
            <Button onClick={() => setShowForm(true)}>Add Address</Button>
          </div>
        </Card>
      )}

      {addresses.length > 0 && !showForm && !editingAddress && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {addresses.map((addr) => (
            <Card key={addr.id}>
              <CardContent>
                <div className="mb-3 flex items-start justify-between">
                  <p className="font-medium text-foreground">
                    {addr.first_name} {addr.last_name}
                    {addr.is_default && (
                      <span className="ml-2 rounded bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                        Default
                      </span>
                    )}
                  </p>
                </div>

                <div className="mb-4 text-sm text-muted-foreground">
                  <p>{addr.address_line1}</p>
                  {addr.address_line2 && <p>{addr.address_line2}</p>}
                  <p>
                    {addr.city}, {addr.state} - {addr.postal_code}
                  </p>
                  <p>Phone: {addr.phone}</p>
                </div>

                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setEditingAddress(addr)}
                  >
                    Edit
                  </Button>
                  <AlertDialog>
                    <AlertDialogTrigger
                      render={<Button variant="destructive" size="sm" loading={deletingId === addr.id} />}
                    >
                      Delete
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete Address</AlertDialogTitle>
                        <AlertDialogDescription>
                          Are you sure you want to delete this address? This action cannot be undone.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                          variant="destructive"
                          onClick={() => handleDelete(addr.id)}
                        >
                          Delete
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
