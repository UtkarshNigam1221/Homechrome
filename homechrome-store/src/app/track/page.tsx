'use client';

import { useState } from 'react';

import Button from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Container } from '@/components/ui/container';
import { PageHeader } from '@/components/ui/page-header';
import Input from '@/components/ui/form-field';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatDateTime as formatDate } from '@/lib/utils';

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
        ROUTES.TRACK(trimmed),
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
    <Container size="narrow" className="max-w-2xl py-12">
      <PageHeader
        title="Track Your Order"
        description="Enter your order number to check the delivery status."
        className="text-center"
      />

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

      {tracking && (
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Order #{tracking.order_number}</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                Current Status:{' '}
                <span className="font-medium text-foreground">
                  {tracking.status}
                </span>
              </p>

              {(tracking.shipping_carrier || tracking.tracking_number) && (
                <div className="mt-4 rounded-lg bg-background p-4">
                  <h3 className="mb-2 text-sm font-semibold text-foreground">
                    Shipment Details
                  </h3>
                  <div className="grid grid-cols-1 gap-2 text-sm sm:grid-cols-3">
                    {tracking.shipping_carrier && (
                      <div>
                        <span className="text-muted-foreground">Courier: </span>
                        <span className="font-medium text-foreground">
                          {tracking.shipping_carrier}
                        </span>
                      </div>
                    )}
                    {tracking.tracking_number && (
                      <div>
                        <span className="text-muted-foreground">AWB: </span>
                        <span className="font-medium text-foreground">
                          {tracking.tracking_number}
                        </span>
                      </div>
                    )}
                    {tracking.estimated_delivery && (
                      <div>
                        <span className="text-muted-foreground">Est. Delivery: </span>
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
            </CardContent>
          </Card>

          {tracking.status_history && tracking.status_history.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Order Timeline</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="relative">
                  <div className="absolute left-3 top-2 bottom-2 w-0.5 bg-border" />

                  <div className="space-y-6">
                    {tracking.status_history.map((entry, idx) => (
                      <div key={idx} className="relative flex gap-4 pl-10">
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
                          <p className="text-xs text-muted-foreground">
                            {formatDate(entry.timestamp)}
                          </p>
                          {entry.description && (
                            <p className="mt-0.5 text-sm text-muted-foreground">
                              {entry.description}
                            </p>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </Container>
  );
}
