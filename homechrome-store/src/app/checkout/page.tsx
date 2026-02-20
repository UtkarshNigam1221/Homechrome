'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useCallback, useEffect, useState } from 'react';

import Button from '@/components/common/Button';
import AddressForm from '@/components/checkout/AddressForm';
import OrderSummary from '@/components/checkout/OrderSummary';
import ShippingOptions from '@/components/checkout/ShippingOptions';
import api from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import {
  Address,
  CartWithItems,
  CheckoutResult,
  CourierOption,
  ServiceabilityResult,
} from '@/types';

type Step = 'address' | 'shipping' | 'review';

export default function CheckoutPage() {
  const router = useRouter();
  const { isAuthenticated, isLoading: authLoading, customer } = useAuthStore();

  const [cart, setCart] = useState<CartWithItems | null>(null);
  const [cartLoading, setCartLoading] = useState(true);

  const [step, setStep] = useState<Step>('address');
  const [selectedAddressId, setSelectedAddressId] = useState<string | null>(null);
  const [showAddressForm, setShowAddressForm] = useState(false);
  const [addressSaving, setAddressSaving] = useState(false);

  const [couriers, setCouriers] = useState<CourierOption[]>([]);
  const [selectedCourierId, setSelectedCourierId] = useState<number | null>(null);
  const [serviceabilityLoading, setServiceabilityLoading] = useState(false);
  const [serviceabilityError, setServiceabilityError] = useState<string | null>(null);

  const [initiating, setInitiating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Redirect unauthenticated users
  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.replace('/login?redirect=/checkout');
    }
  }, [authLoading, isAuthenticated, router]);

  // Fetch cart
  const fetchCart = useCallback(async () => {
    try {
      setCartLoading(true);
      const { data } = await api.get<CartWithItems>('/api/v1/store/cart');
      setCart(data);
      if (!data.items || data.items.length === 0) {
        router.replace('/cart');
      }
    } catch {
      router.replace('/cart');
    } finally {
      setCartLoading(false);
    }
  }, [router]);

  useEffect(() => {
    if (isAuthenticated) {
      fetchCart();
    }
  }, [isAuthenticated, fetchCart]);

  // Pre-select default address
  useEffect(() => {
    if (customer?.addresses && customer.addresses.length > 0 && !selectedAddressId) {
      const defaultAddr = customer.addresses.find((a) => a.is_default);
      setSelectedAddressId(defaultAddr?.id || customer.addresses[0].id);
    }
  }, [customer?.addresses, selectedAddressId]);

  const addresses = customer?.addresses || [];
  const selectedAddress = addresses.find((a) => a.id === selectedAddressId) || null;

  // Check serviceability
  const checkServiceability = useCallback(async () => {
    if (!selectedAddress) return;

    setServiceabilityLoading(true);
    setServiceabilityError(null);
    setCouriers([]);
    setSelectedCourierId(null);

    try {
      const { data } = await api.post<ServiceabilityResult>(
        '/api/v1/store/checkout/serviceability',
        { pincode: selectedAddress.postal_code },
      );

      if (!data.serviceable) {
        setServiceabilityError(
          'Delivery is not available to this PIN code. Please choose a different address.',
        );
        return;
      }

      setCouriers(data.couriers);
      if (data.couriers.length === 1) {
        setSelectedCourierId(data.couriers[0].id);
      }
    } catch {
      setServiceabilityError('Failed to check delivery availability. Please try again.');
    } finally {
      setServiceabilityLoading(false);
    }
  }, [selectedAddress]);

  const handleAddressNext = () => {
    if (!selectedAddressId) return;
    setStep('shipping');
    checkServiceability();
  };

  const handleShippingNext = () => {
    if (!selectedCourierId) return;
    setStep('review');
  };

  const handleAddAddress = async (data: Omit<Address, 'id'>) => {
    setAddressSaving(true);
    try {
      const { data: newAddr } = await api.post<Address>(
        '/api/v1/store/me/addresses',
        data,
      );
      // Refresh customer data to get updated addresses
      await useAuthStore.getState().checkAuth();
      setSelectedAddressId(newAddr.id);
      setShowAddressForm(false);
    } catch {
      setError('Failed to save address. Please try again.');
    } finally {
      setAddressSaving(false);
    }
  };

  const handlePayNow = async () => {
    if (!selectedAddressId) return;

    setInitiating(true);
    setError(null);

    try {
      const payload: { shipping_address_id: string; courier_id?: number } = {
        shipping_address_id: selectedAddressId,
      };
      if (selectedCourierId) {
        payload.courier_id = selectedCourierId;
      }

      const { data } = await api.post<CheckoutResult>(
        '/api/v1/store/checkout/initiate',
        payload,
      );

      // Redirect to PhonePe payment
      window.location.href = data.redirect_url;
    } catch {
      setError('Failed to initiate payment. Please try again.');
      setInitiating(false);
    }
  };

  const selectedCourier = couriers.find((c) => c.id === selectedCourierId) || null;

  // Loading states
  if (authLoading || cartLoading) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="flex items-center justify-center py-20">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      </div>
    );
  }

  if (!cart || !cart.items || cart.items.length === 0) {
    return null;
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
      <h1 className="mb-2 text-2xl font-bold text-foreground sm:text-3xl">Checkout</h1>
      <p className="mb-8 text-sm text-muted">
        Complete your order in a few simple steps.
      </p>

      {/* Progress steps */}
      <div className="mb-8 flex items-center gap-2 text-sm">
        {(['address', 'shipping', 'review'] as const).map((s, i) => (
          <div key={s} className="flex items-center gap-2">
            {i > 0 && <div className="h-px w-6 bg-border sm:w-12" />}
            <button
              type="button"
              onClick={() => {
                if (s === 'address') setStep('address');
                else if (s === 'shipping' && selectedAddressId) {
                  setStep('shipping');
                  if (couriers.length === 0) checkServiceability();
                } else if (s === 'review' && selectedCourierId) setStep('review');
              }}
              className={`flex items-center gap-1.5 rounded-full px-3 py-1 font-medium transition-colors ${
                step === s
                  ? 'bg-primary text-white'
                  : 'bg-white text-muted hover:text-foreground'
              }`}
            >
              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-white/20 text-xs">
                {i + 1}
              </span>
              <span className="hidden sm:inline">
                {s === 'address' ? 'Address' : s === 'shipping' ? 'Shipping' : 'Review'}
              </span>
            </button>
          </div>
        ))}
      </div>

      {error && (
        <div className="mb-6 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
        {/* Main content */}
        <div className="lg:col-span-2">
          {/* Step 1: Address */}
          {step === 'address' && (
            <div className="rounded-lg border border-border bg-white p-6">
              <h2 className="mb-4 text-lg font-semibold text-foreground">
                Shipping Address
              </h2>

              {addresses.length > 0 && !showAddressForm && (
                <div className="space-y-3">
                  {addresses.map((addr) => (
                    <button
                      key={addr.id}
                      type="button"
                      onClick={() => setSelectedAddressId(addr.id)}
                      className={`w-full rounded-lg border p-4 text-left transition-colors ${
                        selectedAddressId === addr.id
                          ? 'border-primary bg-primary/5 ring-1 ring-primary'
                          : 'border-border hover:border-primary/50'
                      }`}
                    >
                      <div className="flex items-start justify-between">
                        <div>
                          <p className="font-medium text-foreground">
                            {addr.first_name} {addr.last_name}
                            {addr.is_default && (
                              <span className="ml-2 rounded bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">
                                Default
                              </span>
                            )}
                          </p>
                          <p className="mt-1 text-sm text-muted">
                            {addr.address_line1}
                            {addr.address_line2 && `, ${addr.address_line2}`}
                          </p>
                          <p className="text-sm text-muted">
                            {addr.city}, {addr.state} - {addr.postal_code}
                          </p>
                          <p className="text-sm text-muted">Phone: {addr.phone}</p>
                        </div>
                        <div
                          className={`flex h-5 w-5 items-center justify-center rounded-full border-2 ${
                            selectedAddressId === addr.id
                              ? 'border-primary'
                              : 'border-muted'
                          }`}
                        >
                          {selectedAddressId === addr.id && (
                            <div className="h-2.5 w-2.5 rounded-full bg-primary" />
                          )}
                        </div>
                      </div>
                    </button>
                  ))}

                  <button
                    type="button"
                    onClick={() => setShowAddressForm(true)}
                    className="mt-2 text-sm font-medium text-primary hover:text-primary-dark"
                  >
                    + Add a new address
                  </button>

                  <div className="mt-4 pt-4">
                    <Button
                      onClick={handleAddressNext}
                      disabled={!selectedAddressId}
                    >
                      Continue to Shipping
                    </Button>
                  </div>
                </div>
              )}

              {(addresses.length === 0 || showAddressForm) && (
                <AddressForm
                  onSubmit={handleAddAddress}
                  onCancel={() => {
                    if (addresses.length > 0) {
                      setShowAddressForm(false);
                    } else {
                      router.push('/cart');
                    }
                  }}
                  loading={addressSaving}
                />
              )}
            </div>
          )}

          {/* Step 2: Shipping */}
          {step === 'shipping' && (
            <div className="rounded-lg border border-border bg-white p-6">
              <h2 className="mb-4 text-lg font-semibold text-foreground">
                Shipping Method
              </h2>

              {selectedAddress && (
                <div className="mb-4 rounded-lg bg-background p-3 text-sm">
                  <p className="font-medium text-foreground">
                    Delivering to: {selectedAddress.first_name}{' '}
                    {selectedAddress.last_name}
                  </p>
                  <p className="text-muted">
                    {selectedAddress.address_line1}, {selectedAddress.city},{' '}
                    {selectedAddress.state} - {selectedAddress.postal_code}
                  </p>
                  <button
                    type="button"
                    onClick={() => setStep('address')}
                    className="mt-1 text-xs font-medium text-primary hover:text-primary-dark"
                  >
                    Change address
                  </button>
                </div>
              )}

              {serviceabilityError && (
                <div className="mb-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
                  {serviceabilityError}
                  <button
                    type="button"
                    onClick={() => setStep('address')}
                    className="mt-2 block font-medium text-red-800 underline"
                  >
                    Choose a different address
                  </button>
                </div>
              )}

              <ShippingOptions
                couriers={couriers}
                selectedId={selectedCourierId}
                onSelect={setSelectedCourierId}
                loading={serviceabilityLoading}
              />

              {!serviceabilityLoading && !serviceabilityError && couriers.length > 0 && (
                <div className="mt-6 flex gap-3">
                  <Button
                    onClick={handleShippingNext}
                    disabled={!selectedCourierId}
                  >
                    Continue to Review
                  </Button>
                  <Button variant="outline" onClick={() => setStep('address')}>
                    Back
                  </Button>
                </div>
              )}
            </div>
          )}

          {/* Step 3: Review */}
          {step === 'review' && (
            <div className="rounded-lg border border-border bg-white p-6">
              <h2 className="mb-4 text-lg font-semibold text-foreground">
                Review Your Order
              </h2>

              {/* Address summary */}
              {selectedAddress && (
                <div className="mb-4 rounded-lg bg-background p-4">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-foreground">
                      Shipping Address
                    </h3>
                    <button
                      type="button"
                      onClick={() => setStep('address')}
                      className="text-xs font-medium text-primary hover:text-primary-dark"
                    >
                      Change
                    </button>
                  </div>
                  <p className="mt-1 text-sm text-muted">
                    {selectedAddress.first_name} {selectedAddress.last_name},{' '}
                    {selectedAddress.address_line1}, {selectedAddress.city},{' '}
                    {selectedAddress.state} - {selectedAddress.postal_code}
                  </p>
                </div>
              )}

              {/* Shipping summary */}
              {selectedCourier && (
                <div className="mb-6 rounded-lg bg-background p-4">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-semibold text-foreground">
                      Shipping Method
                    </h3>
                    <button
                      type="button"
                      onClick={() => setStep('shipping')}
                      className="text-xs font-medium text-primary hover:text-primary-dark"
                    >
                      Change
                    </button>
                  </div>
                  <p className="mt-1 text-sm text-muted">
                    {selectedCourier.name} - Est. {selectedCourier.estimated_days}{' '}
                    {selectedCourier.estimated_days === 1 ? 'day' : 'days'}
                  </p>
                </div>
              )}

              {/* Items */}
              <div className="mb-6 space-y-3">
                <h3 className="text-sm font-semibold text-foreground">Items</h3>
                {cart.items.map((item) => (
                  <div key={item.product_id} className="flex justify-between text-sm">
                    <span className="text-muted">
                      {item.product_name} x {item.quantity}
                    </span>
                    <span className="text-foreground">
                      {`₹${(item.total_price / 100).toLocaleString('en-IN')}`}
                    </span>
                  </div>
                ))}
              </div>

              <div className="flex gap-3">
                <Button onClick={handlePayNow} loading={initiating}>
                  Pay Now
                </Button>
                <Button variant="outline" onClick={() => setStep('shipping')}>
                  Back
                </Button>
              </div>
            </div>
          )}
        </div>

        {/* Sidebar */}
        <div className="lg:col-span-1">
          <div className="sticky top-36">
            <OrderSummary
              items={cart.items}
              subtotal={cart.cart.subtotal}
              shippingCourier={selectedCourier}
            />
            <div className="mt-4 text-center">
              <Link
                href="/cart"
                className="text-sm font-medium text-primary hover:text-primary-dark"
              >
                Back to Cart
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
