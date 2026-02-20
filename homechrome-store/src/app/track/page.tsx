'use client';

import { useState } from 'react';

import Button from '@/components/common/Button';
import Input from '@/components/common/Input';
import api from '@/lib/api';

interface StatusHistoryEntry {
  status: string;
  timestamp: string;
  description?: string;
}

interface TrackingResult {
  order_number: string;
  status: string;
  status_history: StatusHistoryEntry[];
  shipping_carrier?: string;
  tracking_number?: string;
  tracking_url?: string;
  estimated_delivery?: string;
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export default function TrackOrderPage() {
  const [orderNumber, setOrderNumber] = useState('');
  const [tracking, setTracking] = useState<TrackingResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const trimmed = orderNumber.trim();
    if (!trimmed) return;

    setLoading(true);
    setError(null);
    setTracking(null);

    try {
      const { data } = await api.get<TrackingResult>(
        `/api/v1/store/track/${encodeURIComponent(trimmed)}`,
      );
      setTracking(data);
    } catch {
      setError(
        'Order not found. Please check the order number and try again.',
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl px-4 py-12 sm:px-6 lg:px-8">
      <div className="mb-8 text-center">
        <h1 className="mb-2 text-2xl font-bold text-foreground sm:text-3xl">
          Track Your Order
        </h1>
        <p className="text-muted">
          Enter your order number to check the delivery status.
        </p>
      </div>

      <form onSubmit={handleSubmit} className="mb-8">
        <div className="flex gap-3">
          <div className="flex-1">
            <Input
              value={orderNumber}
              onChange={(e) => setOrderNumber(e.target.value)}
              placeholder="Enter your order number (e.g., HC-20260220-XXXX)"
              error={error || undefined}
            />
          </div>
          <Button type="submit" loading={loading} className="flex-shrink-0">
            Track
          </Button>
        </div>
      </form>

      {/* Results */}
      {tracking && (
        <div className="space-y-6">
          {/* Order info */}
          <div className="rounded-lg border border-border bg-white p-6">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 className="text-lg font-semibold text-foreground">
                  Order #{tracking.order_number}
                </h2>
                <p className="text-sm text-muted">
                  Current Status:{' '}
                  <span className="font-medium text-foreground">
                    {tracking.status}
                  </span>
                </p>
              </div>
            </div>

            {/* Shipment info */}
            {(tracking.shipping_carrier || tracking.tracking_number) && (
              <div className="mt-4 rounded-lg bg-background p-4">
                <h3 className="mb-2 text-sm font-semibold text-foreground">
                  Shipment Details
                </h3>
                <div className="grid grid-cols-1 gap-2 text-sm sm:grid-cols-3">
                  {tracking.shipping_carrier && (
                    <div>
                      <span className="text-muted">Courier: </span>
                      <span className="font-medium text-foreground">
                        {tracking.shipping_carrier}
                      </span>
                    </div>
                  )}
                  {tracking.tracking_number && (
                    <div>
                      <span className="text-muted">AWB: </span>
                      <span className="font-medium text-foreground">
                        {tracking.tracking_number}
                      </span>
                    </div>
                  )}
                  {tracking.estimated_delivery && (
                    <div>
                      <span className="text-muted">Est. Delivery: </span>
                      <span className="font-medium text-foreground">
                        {formatDate(tracking.estimated_delivery)}
                      </span>
                    </div>
                  )}
                </div>
                {tracking.tracking_url && (
                  <a
                    href={tracking.tracking_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-2 inline-block text-sm font-medium text-primary hover:text-primary-dark"
                  >
                    Track on courier website &rarr;
                  </a>
                )}
              </div>
            )}
          </div>

          {/* Status Timeline */}
          {tracking.status_history && tracking.status_history.length > 0 && (
            <div className="rounded-lg border border-border bg-white p-6">
              <h3 className="mb-4 text-sm font-semibold text-foreground">
                Order Timeline
              </h3>
              <div className="relative">
                {/* Vertical line */}
                <div className="absolute left-3 top-2 bottom-2 w-0.5 bg-border" />

                <div className="space-y-6">
                  {tracking.status_history.map((entry, idx) => (
                    <div key={idx} className="relative flex gap-4 pl-10">
                      {/* Dot */}
                      <div
                        className={`absolute left-1 top-1 h-5 w-5 rounded-full border-2 ${
                          idx === 0
                            ? 'border-primary bg-primary'
                            : 'border-border bg-white'
                        }`}
                      >
                        {idx === 0 && (
                          <div className="flex h-full w-full items-center justify-center">
                            <div className="h-1.5 w-1.5 rounded-full bg-white" />
                          </div>
                        )}
                      </div>

                      <div>
                        <p className="font-medium text-foreground">
                          {entry.status}
                        </p>
                        <p className="text-xs text-muted">
                          {formatDate(entry.timestamp)}
                        </p>
                        {entry.description && (
                          <p className="mt-0.5 text-sm text-muted">
                            {entry.description}
                          </p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
