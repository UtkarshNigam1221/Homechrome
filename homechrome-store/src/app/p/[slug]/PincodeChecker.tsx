'use client';

import { useRef, useState } from 'react';

import { Alert } from '@/components/ui/alert';
import Button from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { shippingApi } from '@/lib/shipping-api';
import type { ServiceabilityResult } from '@/types';

export default function PincodeChecker() {
  const [pin, setPin] = useState('');
  const [result, setResult] = useState<ServiceabilityResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const cacheRef = useRef<Map<string, ServiceabilityResult>>(new Map());

  const onCheck = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setResult(null);
    if (!/^[1-9][0-9]{5}$/.test(pin)) {
      setError('Enter a valid 6-digit pincode');
      return;
    }
    const cached = cacheRef.current.get(pin);
    if (cached) {
      setResult(cached);
      return;
    }
    setLoading(true);
    try {
      const r = await shippingApi.checkPincode(pin);
      cacheRef.current.set(pin, r);
      setResult(r);
    } catch {
      setError('Could not check pincode. Try again.');
    } finally {
      setLoading(false);
    }
  };

  const firstCourier = result?.couriers?.[0];

  return (
    <div className="rounded-lg border border-border bg-white p-4">
      <h3 className="mb-2 text-sm font-semibold text-foreground">
        Check delivery
      </h3>
      <form onSubmit={onCheck} className="flex gap-2">
        <Input
          type="text"
          inputMode="numeric"
          maxLength={6}
          value={pin}
          onChange={(e) => setPin(e.target.value.replace(/\D/g, ''))}
          placeholder="6-digit pincode"
          aria-label="Pincode"
          aria-invalid={!!error}
          className="flex-1"
        />
        <Button type="submit" loading={loading} size="default">
          Check
        </Button>
      </form>
      {error && (
        <div className="mt-2">
          <Alert variant="error">{error}</Alert>
        </div>
      )}
      {result && (
        <div className="mt-3">
          {result.serviceable ? (
            <Alert variant="success">
              <span aria-hidden="true">✓ </span>Delivery available
              {firstCourier?.estimated_days
                ? ` in ${firstCourier.estimated_days} ${
                    firstCourier.estimated_days === 1 ? 'day' : 'days'
                  }`
                : null}
            </Alert>
          ) : (
            <Alert variant="warning">
              Sorry, we do not deliver to {pin} right now.
            </Alert>
          )}
        </div>
      )}
    </div>
  );
}
