import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Edit, RefreshCw } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { shippingApi } from '@/features/shipping/api';
import type { ShippingRate } from '@/features/shipping/types';
import { getErrorMessage } from '@/shared/api/client';
import { PageHeader } from '@/shared/components/layout';
import {
  Badge,
  Button,
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
import { formatPaiseExact } from '@/shared/utils/currency';

import { RateEditModal } from './RateEditModal';

function sortRates(rates: ShippingRate[]): ShippingRate[] {
  return [...rates].sort((a, b) => {
    if (a.zone !== b.zone) return a.zone.localeCompare(b.zone);
    return a.weight_slab_grams - b.weight_slab_grams;
  });
}

export function RatesPage() {
  const queryClient = useQueryClient();
  const [editingRate, setEditingRate] = useState<ShippingRate | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['shipping', 'rates'],
    queryFn: shippingApi.listRates,
  });

  const refreshMutation = useMutation({
    mutationFn: shippingApi.triggerRateRefresh,
    onSuccess: (res) => {
      const label = res.status === 'refresh_queued' ? 'queued' : 'completed';
      toast.success(`Rate refresh ${label}`);
      queryClient.invalidateQueries({ queryKey: ['shipping', 'rates'] });
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const rates = sortRates(data ?? []);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Shipping Rates"
        subtitle="Delhivery rate matrix by zone and weight slab. Prices shown in INR."
        action={
          <Button
            leftIcon={<RefreshCw className="w-4 h-4" />}
            onClick={() => refreshMutation.mutate()}
            loading={refreshMutation.isPending}
          >
            Refresh from Delhivery
          </Button>
        }
      />

      <Card padding="none">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Zone</TableHead>
              <TableHead>Weight slab</TableHead>
              <TableHead>Prepaid</TableHead>
              <TableHead>COD</TableHead>
              <TableHead>RTO</TableHead>
              <TableHead>Source</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableLoading rows={6} colSpan={7} />
            ) : rates.length === 0 ? (
              <TableEmpty
                colSpan={7}
                message="No shipping rates loaded yet"
                action={
                  <Button
                    leftIcon={<RefreshCw className="w-4 h-4" />}
                    onClick={() => refreshMutation.mutate()}
                  >
                    Refresh from Delhivery
                  </Button>
                }
              />
            ) : (
              rates.map((rate) => (
                <TableRow key={`${rate.zone}-${rate.weight_slab_grams}`}>
                  <TableCell>
                    <span className="font-mono font-medium">{rate.zone}</span>
                  </TableCell>
                  <TableCell>{rate.weight_slab_grams} g</TableCell>
                  <TableCell>{formatPaiseExact(rate.prepaid_paise)}</TableCell>
                  <TableCell>{formatPaiseExact(rate.cod_paise)}</TableCell>
                  <TableCell>{formatPaiseExact(rate.rto_paise)}</TableCell>
                  <TableCell>
                    <Badge variant={rate.source === 'manual_override' ? 'warning' : 'info'}>
                      {rate.source === 'manual_override' ? 'Manual' : 'API'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditingRate(rate)}
                        title="Edit rate"
                      >
                        <Edit className="w-4 h-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Card>

      <RateEditModal
        isOpen={!!editingRate}
        onClose={() => setEditingRate(null)}
        rate={editingRate}
      />
    </div>
  );
}
