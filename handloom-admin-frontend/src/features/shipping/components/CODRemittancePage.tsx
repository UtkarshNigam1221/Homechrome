import { useQuery } from '@tanstack/react-query';
import { clsx } from 'clsx';
import { format } from 'date-fns';
import { useState } from 'react';

import { shippingApi } from '@/features/shipping/api';
import { COD_REMITTANCE_STATUSES, type CODRemittanceStatus } from '@/features/shipping/types';
import { PageHeader } from '@/shared/components/layout';
import {
  Badge,
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
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency } from '@/shared/utils/currency';

import { CODRemittanceDetail } from './CODRemittanceDetail';

export function CODRemittancePage() {
  const [status, setStatus] = useState<CODRemittanceStatus>('RECONCILED');
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['shipping', 'cod-remittances', status],
    queryFn: () => shippingApi.listCODRemittances(status),
  });

  const remittances = data ?? [];

  return (
    <div className="space-y-6">
      <PageHeader
        title="COD Remittance"
        subtitle="Daily COD payouts pulled from Delhivery and reconciled against orders."
      />

      <Card padding="sm">
        <div className="flex flex-wrap gap-2">
          {COD_REMITTANCE_STATUSES.map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => setStatus(s)}
              className={clsx(
                'px-3 py-1.5 rounded-full text-xs font-medium ring-1 ring-inset transition-colors',
                s === status
                  ? 'bg-primary-50 text-primary-700 ring-primary-600/30'
                  : 'bg-white text-gray-600 ring-gray-200 hover:bg-gray-50'
              )}
            >
              {s}
            </button>
          ))}
        </div>
      </Card>

      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Remittance ref</TableHead>
              <TableHead>Amount</TableHead>
              <TableHead>Remitted at</TableHead>
              <TableHead>Entries</TableHead>
              <TableHead>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={5} colSpan={5} />
            ) : remittances.length === 0 ? (
              <TableEmpty colSpan={5} message={`No remittances with status ${status}`} />
            ) : (
              remittances.map((rem) => (
                <TableRow key={rem.id} clickable onClick={() => setSelectedId(rem.id)}>
                  <TableCell>
                    <span className="font-mono text-sm">{rem.remittance_ref}</span>
                  </TableCell>
                  <TableCell>{formatCurrency(rem.amount_paise)}</TableCell>
                  <TableCell>
                    {rem.remitted_at
                      ? format(new Date(rem.remitted_at), 'MMM d, yyyy h:mm a')
                      : '—'}
                  </TableCell>
                  <TableCell>{rem.entries?.length ?? 0}</TableCell>
                  <TableCell>
                    <Badge variant={getStatusBadgeVariant(rem.status)}>{rem.status}</Badge>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>

      <CODRemittanceDetail
        isOpen={!!selectedId}
        onClose={() => setSelectedId(null)}
        remittanceId={selectedId}
      />
    </div>
  );
}
