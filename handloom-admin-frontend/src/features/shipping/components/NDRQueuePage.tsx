import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';

import { shippingApi } from '@/features/shipping/api';
import { PageHeader } from '@/shared/components/layout';
import {
  Card,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '@/shared/components/ui';

import { NDRActionMenu } from './NDRActionMenu';
import { PriorityBadge } from './PriorityBadge';

export function NDRQueuePage() {
  const { data, isLoading } = useQuery({
    queryKey: ['shipping', 'ndr-queue'],
    queryFn: shippingApi.listNDRQueue,
  });

  const shipments = data ?? [];

  return (
    <div className="space-y-6">
      <PageHeader
        title="NDR Queue"
        subtitle="Shipments escalated after repeated non-delivery attempts. Triage to re-attempt, contact, or return to origin."
      />

      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>AWB</TableHead>
              <TableHead>Order</TableHead>
              <TableHead>Priority</TableHead>
              <TableHead>Attempts</TableHead>
              <TableHead>Last NDR reason</TableHead>
              <TableHead>Last NDR at</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={7} />
            ) : shipments.length === 0 ? (
              <TableEmpty colSpan={7} message="No NDR-escalated shipments. Nice." />
            ) : (
              shipments.map((s) => (
                <TableRow key={s.id}>
                  <TableCell>
                    <span className="font-mono text-sm">{s.awb_number || '—'}</span>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs break-all">{s.order_id}</span>
                  </TableCell>
                  <TableCell>
                    <PriorityBadge priority={s.priority} />
                  </TableCell>
                  <TableCell>{s.ndr_count}</TableCell>
                  <TableCell className="max-w-xs truncate" title={s.last_ndr_reason}>
                    {s.last_ndr_reason || '—'}
                  </TableCell>
                  <TableCell>
                    {s.last_ndr_at ? format(new Date(s.last_ndr_at), 'MMM d, h:mm a') : '—'}
                  </TableCell>
                  <TableCell className="text-right">
                    {s.awb_number ? (
                      <NDRActionMenu awb={s.awb_number} />
                    ) : (
                      <span className="text-xs text-gray-400" title="No AWB; cannot act">
                        —
                      </span>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>
    </div>
  );
}
