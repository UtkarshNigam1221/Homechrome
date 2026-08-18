import { useQuery } from '@tanstack/react-query';
import { ExternalLink } from 'lucide-react';
import { useMemo } from 'react';
import { Link } from 'react-router-dom';

import { openReservationIDs } from '@/features/inventory/lib/openReservations';
import type { TransactionType } from '@/features/inventory/types';
import { productsApi } from '@/features/products/api';
import type { Product } from '@/features/products/types';
import {
  Badge,
  Modal,
  Pagination,
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableLoading,
  TableRow,
} from '@/shared/components/ui';
import { ROUTES } from '@/shared/constants/routes';
import { useCursorPagination } from '@/shared/hooks';

export interface InventoryLedgerModalProps {
  isOpen: boolean;
  onClose: () => void;
  product: Product | null;
}

type BadgeVariant = 'success' | 'warning' | 'danger' | 'info' | 'gray' | 'primary';

// previous_qty/new_qty track reserved_qty for RESERVE and RELEASE, but quantity
// for every other type. One "Before → After" header would mean two things, so
// each row names its own counter.
const MOVEMENTS: Record<
  TransactionType,
  { label: string; variant: BadgeVariant; counter: string }
> = {
  ADD: { label: 'Restocked', variant: 'success', counter: 'on hand' },
  REMOVE: { label: 'Removed', variant: 'danger', counter: 'on hand' },
  ADJUST: { label: 'Recounted', variant: 'gray', counter: 'on hand' },
  RESERVE: { label: 'Reserved', variant: 'warning', counter: 'reserved' },
  RELEASE: { label: 'Released', variant: 'info', counter: 'reserved' },
  COMMIT: { label: 'Dispatched', variant: 'primary', counter: 'on hand' },
  RETURN: { label: 'Returned', variant: 'success', counter: 'on hand' },
};

function formatWhen(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function InventoryLedgerModal({ isOpen, onClose, product }: InventoryLedgerModalProps) {
  const { limit, cursor, hasPrevious, goToNextPage, goToPreviousPage, changeLimit } =
    useCursorPagination(10);

  const productID = product?.id ?? '';

  const { data, isLoading } = useQuery({
    queryKey: ['inventory-transactions', productID, { limit, cursor }],
    queryFn: () => productsApi.getInventoryTransactions(productID, { limit, cursor }),
    enabled: isOpen && productID !== '',
  });

  const rows = useMemo(() => data?.items ?? [], [data]);
  const openReservations = useMemo(() => openReservationIDs(rows), [rows]);
  const pagination = data?.pagination;

  if (!product) return null;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Stock history — ${product.name}`}
      description={`Every movement recorded for ${product.sku}, newest first.`}
      size="xl"
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>When</TableHead>
            <TableHead>Movement</TableHead>
            <TableHead>Change</TableHead>
            <TableHead>Before → after</TableHead>
            <TableHead>Reference</TableHead>
            <TableHead>By</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableLoading colSpan={6} />
          ) : rows.length === 0 ? (
            <TableEmpty
              colSpan={6}
              message="No stock movements yet"
              description="Reservations, dispatches and restocks appear here as they happen."
            />
          ) : (
            rows.map((row) => {
              const movement = MOVEMENTS[row.type];
              const orderID = row.reference_type === 'ORDER' ? row.reference_id : undefined;
              const isOpenReservation =
                row.type === 'RESERVE' && row.reference_id
                  ? openReservations.has(row.reference_id)
                  : false;

              return (
                <TableRow key={row.id}>
                  <TableCell>
                    <span className="text-sm whitespace-nowrap text-gray-600">
                      {formatWhen(row.created_at)}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Badge variant={movement.variant}>{movement.label}</Badge>
                      {isOpenReservation && (
                        <Badge variant="warning" dot>
                          Still held
                        </Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-sm">{row.quantity}</span>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-sm">
                      {row.previous_qty} → {row.new_qty}
                    </span>
                    <span className="ml-2 text-xs text-gray-500">{movement.counter}</span>
                  </TableCell>
                  <TableCell>
                    {orderID ? (
                      <Link
                        to={ROUTES.ORDERS.DETAIL(orderID)}
                        className="inline-flex items-center gap-1 font-mono text-sm text-blue-600 hover:underline"
                      >
                        {orderID}
                        <ExternalLink className="h-3 w-3" />
                      </Link>
                    ) : (
                      <span className="text-sm text-gray-600">{row.reason || '—'}</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-gray-600">{row.created_by || 'System'}</span>
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {openReservations.size > 0 && (
        <p className="mt-3 text-xs text-gray-500">
          {openReservations.size} order
          {openReservations.size === 1 ? '' : 's'} still holding stock on this page. A reservation
          settles when the same order dispatches or releases it, so one that never does is stock
          nobody can sell. Only movements on this page are compared.
        </p>
      )}

      <div className="mt-2 border-t border-gray-200">
        <Pagination
          hasMore={pagination?.has_more ?? false}
          hasPrevious={hasPrevious}
          perPage={limit}
          onNextPage={() => pagination?.next_cursor && goToNextPage(pagination.next_cursor)}
          onPreviousPage={goToPreviousPage}
          onPerPageChange={changeLimit}
          itemCount={rows.length}
        />
      </div>
    </Modal>
  );
}
