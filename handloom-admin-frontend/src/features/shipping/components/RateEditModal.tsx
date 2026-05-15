import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { shippingApi } from '@/features/shipping/api';
import type { ShippingRate, UpdateRateRequest } from '@/features/shipping/types';
import { getErrorMessage } from '@/shared/api/client';
import { Button, Input, Modal } from '@/shared/components/ui';

const rateSchema = z.object({
  prepaid_paise: z.number().int('Must be a whole number').min(0, 'Must be non-negative'),
  cod_paise: z.number().int('Must be a whole number').min(0, 'Must be non-negative'),
  rto_paise: z.number().int('Must be a whole number').min(0, 'Must be non-negative'),
});

type RateFormData = z.infer<typeof rateSchema>;

interface RateEditModalProps {
  isOpen: boolean;
  onClose: () => void;
  rate: ShippingRate | null;
}

interface RateEditFormProps {
  rate: ShippingRate;
  onClose: () => void;
}

function RateEditForm({ rate, onClose }: Readonly<RateEditFormProps>) {
  const queryClient = useQueryClient();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RateFormData>({
    resolver: zodResolver(rateSchema),
    defaultValues: {
      prepaid_paise: rate.prepaid_paise,
      cod_paise: rate.cod_paise,
      rto_paise: rate.rto_paise,
    },
  });

  const mutation = useMutation({
    mutationFn: (body: UpdateRateRequest) =>
      shippingApi.updateRate(rate.zone, rate.weight_slab_grams, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shipping', 'rates'] });
      toast.success('Rate updated');
      onClose();
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const onSubmit = (data: RateFormData) => {
    mutation.mutate({
      prepaid_paise: data.prepaid_paise,
      cod_paise: data.cod_paise,
      rto_paise: data.rto_paise,
    });
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <Input
        label="Prepaid (paise)"
        type="number"
        step="1"
        min="0"
        error={errors.prepaid_paise?.message}
        required
        {...register('prepaid_paise', { valueAsNumber: true })}
      />
      <Input
        label="COD (paise)"
        type="number"
        step="1"
        min="0"
        error={errors.cod_paise?.message}
        required
        {...register('cod_paise', { valueAsNumber: true })}
      />
      <Input
        label="RTO (paise)"
        type="number"
        step="1"
        min="0"
        error={errors.rto_paise?.message}
        required
        {...register('rto_paise', { valueAsNumber: true })}
      />

      <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
        <Button variant="secondary" onClick={onClose} disabled={mutation.isPending}>
          Cancel
        </Button>
        <Button type="submit" loading={mutation.isPending}>
          Save Rate
        </Button>
      </div>
    </form>
  );
}

export function RateEditModal({ isOpen, onClose, rate }: Readonly<RateEditModalProps>) {
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={rate ? `Edit Rate: ${rate.zone} / ${rate.weight_slab_grams}g` : 'Edit Rate'}
      description="Saving marks this row as a manual override."
      size="sm"
    >
      {rate && (
        <RateEditForm
          key={`${rate.zone}-${rate.weight_slab_grams}`}
          rate={rate}
          onClose={onClose}
        />
      )}
    </Modal>
  );
}
