'use client';

import { isAxiosError } from 'axios';
import { useState } from 'react';

import TrackingTimeline from '@/components/tracking/TrackingTimeline';
import Button from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Container } from '@/components/ui/container';
import FormField from '@/components/ui/form-field';
import { PageHeader } from '@/components/ui/page-header';
import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatDateTime } from '@/lib/utils';
import type { TrackingScan } from '@/types/shipping';

interface StatusHistoryEntry {
  status: string;
  timestamp: string;
  description?: string;
  location?: string;
  note?: string;
}

interface ShipmentInfo {
  awb_number?: string;
  courier_name?: string;
  tracking_url?: string;
  status?: string;
  estimated_delivery?: string;
}

interface TrackingResult {
  order_number: string;
  status: string;
  status_history: StatusHistoryEntry[];
  shipment?: ShipmentInfo;
  // legacy/top-level fields some responses may use
  shipping_carrier?: string;
  tracking_number?: string;
  tracking_url?: string;
  estimated_delivery?: string;
}

function toTimelineScans(entries: StatusHistoryEntry[]): TrackingScan[] {
  // Latest first
  return [...entries]
    .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    .map((entry) => ({
      status: entry.status,
      timestamp: entry.timestamp,
      location: entry.location,
      description: entry.description || entry.note,
    }));
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
    } catch (err) {
      const status = isAxiosError(err) ? err.response?.status : undefined;
      if (status === 404) {
        setError('Order not found. Check the order number and try again.');
      } else {
        setError('Could not fetch tracking. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  const carrier = tracking?.shipment?.courier_name ?? tracking?.shipping_carrier;
  const awb = tracking?.shipment?.awb_number ?? tracking?.tracking_number;
  const trackingUrl = tracking?.shipment?.tracking_url ?? tracking?.tracking_url;
  const eta = tracking?.shipment?.estimated_delivery ?? tracking?.estimated_delivery;
  const scans = tracking ? toTimelineScans(tracking.status_history || []) : [];

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
            <FormField
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

              {(carrier || awb) && (
                <div className="mt-4 rounded-lg bg-background p-4">
                  <h3 className="mb-2 text-sm font-semibold text-foreground">
                    Shipment Details
                  </h3>
                  <div className="grid grid-cols-1 gap-2 text-sm sm:grid-cols-3">
                    {carrier && (
                      <div>
                        <span className="text-muted-foreground">Courier: </span>
                        <span className="font-medium text-foreground">
                          {carrier}
                        </span>
                      </div>
                    )}
                    {awb && (
                      <div>
                        <span className="text-muted-foreground">AWB: </span>
                        <span className="font-medium text-foreground">
                          {awb}
                        </span>
                      </div>
                    )}
                    {eta && (
                      <div>
                        <span className="text-muted-foreground">Est. Delivery: </span>
                        <span className="font-medium text-foreground">
                          {formatDateTime(eta)}
                        </span>
                      </div>
                    )}
                  </div>
                  {trackingUrl && (
                    <a
                      href={trackingUrl}
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

          {scans.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Tracking Timeline</CardTitle>
              </CardHeader>
              <CardContent>
                <TrackingTimeline scans={scans} currentStatus={tracking.status} />
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </Container>
  );
}
