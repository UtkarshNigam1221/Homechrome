import { useQuery } from '@tanstack/react-query';
import { ExternalLink } from 'lucide-react';
import { useMemo } from 'react';
import { Link } from 'react-router-dom';

import {
  balancesAfter,
  movementEffect,
  recordedCounter,
} from '@/features/inventory/lib/movementEffects';
import { openReservationIDs } from '@/features/inventory/lib/openReservations';
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
import { useCursorPagination } from '@/shared/hooks';

export interface InventoryLedgerModalProps {
  isOpen: boolean;
  onClose: () => void;
  product: Product | null;
}

function formatWhen(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// A signed count, coloured by direction, so a row reads as stock in or out
// without decoding the movement name first.
function Delta({ value }: { value: number | undefined }) {
  if (value === undefined) {
    return <span className="text-gray-300">—</span>;
  }

  return (
    <span
      className={`font-mono text-sm font-medium ${value < 0 ? 'text-red-600' : 'text-emerald-600'}`}
    >
      {value > 0 ? '+' : '−'}
      {Math.abs(value)}
    </span>
  );
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
  const balances = useMemo(
    () =>
      balancesAfter(rows, {
        onHand: product?.quantity ?? 0,
        reserved: product?.reserved_qty ?? 0,
      }),
    [rows, product]
  );
  const pagination = data?.pagination;

  if (!product) return null;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Stock history — ${product.name}`}
      description={`${product.sku} · on hand ${product.quantity ?? 0}, of which ${product.reserved_qty ?? 0} reserved`}
      size="full"
    >
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>When</TableHead>
            <TableHead>Movement</TableHead>
            <TableHead>On hand</TableHead>
            <TableHead>Reserved</TableHead>
            <TableHead>Balance after</TableHead>
            <TableHead>Why</TableHead>
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
            rows.map((row, index) => {
              const effect = movementEffect(row);
              const balance = balances[index];
              const counter = recordedCounter(row.type);
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
                      <Badge variant={effect.variant}>{effect.label}</Badge>
                      {isOpenReservation && (
                        <Badge variant="warning" dot>
                          Still held
                        </Badge>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Delta value={effect.onHand} />
                  </TableCell>
                  <TableCell>
                    <Delta value={effect.reserved} />
                  </TableCell>
                  <TableCell>
                    {balance ? (
                      <div className="space-y-0.5">
                        <div className="flex items-baseline gap-2">
                          <span className="w-8 text-right font-mono text-sm">{balance.onHand}</span>
                          <span className="text-xs text-gray-500">on hand</span>
                        </div>
                        <div className="flex items-baseline gap-2">
                          <span className="w-8 text-right font-mono text-sm text-gray-600">
                            {balance.reserved}
                          </span>
                          <span className="text-xs text-gray-500">reserved</span>
                        </div>
                      </div>
                    ) : (
                      <div className="flex items-baseline gap-2">
                        <span className="w-8 text-right font-mono text-sm">{row.new_qty}</span>
                        <span className="text-xs text-gray-500">
                          {counter === 'reserved' ? 'reserved' : 'on hand'}
                        </span>
                      </div>
                    )}
                  </TableCell>
                  <TableCell>
                    {orderID ? (
                      <Link
                        to={`/orders/${orderID}`}
                        className="inline-flex items-center gap-1 font-mono text-sm text-blue-600 hover:underline"
                      >
                        {orderID}
                        <ExternalLink className="h-3 w-3" />
                      </Link>
                    ) : (
                      <div>
                        <p className="text-sm text-gray-700">{row.reason || '—'}</p>
                        {row.created_by && (
                          <p className="text-xs text-gray-500">{row.created_by}</p>
                        )}
                      </div>
                    )}
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>

      {openReservations.size > 0 && (
        <p className="mt-3 text-xs text-gray-500">
          {openReservations.size} order{openReservations.size === 1 ? '' : 's'} on this page
          reserved stock and never dispatched or released it, so those units are held against
          nothing. Only movements on this page are compared.
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
