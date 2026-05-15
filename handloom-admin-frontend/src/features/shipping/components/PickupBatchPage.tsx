import { useMutation } from '@tanstack/react-query';
import { CalendarClock, PackageCheck, PackageX, Play } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { shippingApi } from '@/features/shipping/api';
import type { BatchResult } from '@/features/shipping/types';
import { getErrorMessage } from '@/shared/api/client';
import { PageHeader } from '@/shared/components/layout';
import { Badge, Button, Card } from '@/shared/components/ui';

// The pickup batch cron runs daily MON-FRI at 17:00 IST. Surfaced as static
// reference text so operators know when the next automated run is.
const NEXT_SCHEDULED = 'Daily Mon-Fri, 17:00 IST';

export function PickupBatchPage() {
  const [lastResult, setLastResult] = useState<BatchResult | null>(null);

  const runMutation = useMutation({
    mutationFn: shippingApi.runPickupBatch,
    onSuccess: (res) => {
      setLastResult(res);
      toast.success(
        res.shipment_count > 0
          ? `Batch ran: ${res.shipment_count} shipments processed`
          : 'No shipments queued for pickup'
      );
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  return (
    <div className="space-y-6">
      <PageHeader
        title="Pickup Batches"
        subtitle="Manifest all CREATED shipments and schedule a pickup with Delhivery."
        action={
          <Button
            leftIcon={<Play className="w-4 h-4" />}
            onClick={() => runMutation.mutate()}
            loading={runMutation.isPending}
          >
            Run pickup batch now
          </Button>
        }
      />

      <Card>
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-primary-50 flex items-center justify-center">
            <CalendarClock className="w-5 h-5 text-primary-600" />
          </div>
          <div>
            <p className="text-sm text-gray-500">Next scheduled run</p>
            <p className="font-medium">{NEXT_SCHEDULED}</p>
          </div>
        </div>
      </Card>

      {lastResult && (
        <Card>
          <h2 className="text-lg font-semibold mb-4">Last run</h2>
          <dl className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div>
              <dt className="text-gray-500">Manifest ID</dt>
              <dd className="font-mono break-all">{lastResult.manifest_id || '—'}</dd>
            </div>
            <div>
              <dt className="text-gray-500">Shipments attempted</dt>
              <dd className="font-medium">{lastResult.shipment_count}</dd>
            </div>
            <div>
              <dt className="text-gray-500 flex items-center gap-1">
                <PackageCheck className="w-4 h-4 text-emerald-600" /> Marked manifested
              </dt>
              <dd>
                <Badge variant="success">{lastResult.shipment_marked_ids?.length ?? 0}</Badge>
              </dd>
            </div>
            <div>
              <dt className="text-gray-500 flex items-center gap-1">
                <PackageX className="w-4 h-4 text-red-600" /> Failed
              </dt>
              <dd>
                <Badge variant={lastResult.failed_shipment_ids?.length ? 'danger' : 'gray'}>
                  {lastResult.failed_shipment_ids?.length ?? 0}
                </Badge>
              </dd>
            </div>
          </dl>

          {lastResult.failed_shipment_ids && lastResult.failed_shipment_ids.length > 0 && (
            <div className="mt-4 rounded-lg border border-red-100 bg-red-50 p-3 text-sm">
              <p className="font-medium text-red-700 mb-1">Failed shipment IDs</p>
              <ul className="list-disc list-inside text-red-700/90 font-mono text-xs space-y-1">
                {lastResult.failed_shipment_ids.map((id) => (
                  <li key={id}>{id}</li>
                ))}
              </ul>
              <p className="mt-2 text-xs text-red-700/80">
                These shipments were manifested with Delhivery but the local status update failed.
                Reconcile manually before the next batch.
              </p>
            </div>
          )}
        </Card>
      )}
    </div>
  );
}
