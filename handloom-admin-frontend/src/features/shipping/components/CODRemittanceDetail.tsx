import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';

import { shippingApi } from '@/features/shipping/api';
import { PageLoading } from '@/shared/components/loading';
import {
  Badge,
  Button,
  Modal,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableRow,
} from '@/shared/components/ui';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency } from '@/shared/utils/currency';

interface CODRemittanceDetailProps {
  isOpen: boolean;
  onClose: () => void;
  remittanceId: string | null;
}

export function CODRemittanceDetail({ isOpen, onClose, remittanceId }: CODRemittanceDetailProps) {
  const { data: remittance, isLoading } = useQuery({
    queryKey: ['shipping', 'cod-remittance', remittanceId],
    queryFn: () => shippingApi.getCODRemittance(remittanceId ?? ''),
    enabled: isOpen && !!remittanceId,
  });

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={remittance ? `Remittance ${remittance.remittance_ref}` : 'Remittance'}
      size="xl"
    >
      {isLoading || !remittance ? (
        <PageLoading message="Loading remittance…" />
      ) : (
        <div className="space-y-4">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <p className="text-gray-500">Status</p>
              <Badge variant={getStatusBadgeVariant(remittance.status)}>{remittance.status}</Badge>
            </div>
            <div>
              <p className="text-gray-500">Total</p>
              <p className="font-medium">{formatCurrency(remittance.amount_paise)}</p>
            </div>
            <div>
              <p className="text-gray-500">Remitted at</p>
              <p>{format(new Date(remittance.remitted_at), 'MMM d, yyyy h:mm a')}</p>
            </div>
            <div>
              <p className="text-gray-500">Bank ref</p>
              <p className="font-mono text-xs break-all">{remittance.bank_ref || '—'}</p>
            </div>
          </div>

          <div className="border border-gray-200 rounded-lg overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>AWB</TableHead>
                  <TableHead>Order</TableHead>
                  <TableHead>Amount</TableHead>
                  <TableHead>Matched</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {remittance.entries.length === 0 ? (
                  <TableEmpty colSpan={4} message="No entries on this remittance" />
                ) : (
                  remittance.entries.map((entry) => (
                    <TableRow key={entry.awb}>
                      <TableCell>
                        <span className="font-mono text-sm">{entry.awb}</span>
                      </TableCell>
                      <TableCell>
                        {entry.order_id ? (
                          <span className="font-mono text-xs">{entry.order_id}</span>
                        ) : (
                          <span className="text-gray-400">unmatched</span>
                        )}
                      </TableCell>
                      <TableCell>{formatCurrency(entry.amount_paise)}</TableCell>
                      <TableCell>
                        <Badge variant={entry.matched ? 'success' : 'danger'}>
                          {entry.matched ? 'Matched' : 'Unmatched'}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          <div className="flex justify-end pt-4 border-t border-gray-200">
            <Button variant="secondary" onClick={onClose}>
              Close
            </Button>
          </div>
        </div>
      )}
    </Modal>
  );
}
