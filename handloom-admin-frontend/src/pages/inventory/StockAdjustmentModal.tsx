import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowDownCircle, ArrowUpCircle } from 'lucide-react';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { getErrorMessage, inventoryApi } from '../../api';
import { Button, Input, Modal, Select } from '../../components/common';
import type { Product } from '../../types';

const stockAdjustmentSchema = z.object({
  type: z.enum(['ADD', 'REMOVE', 'ADJUST']),
  quantity: z.number().min(1, 'Quantity must be at least 1'),
  reason: z
    .string()
    .min(1, 'Reason is required')
    .max(500, 'Reason must be less than 500 characters'),
});

type StockAdjustmentFormData = z.infer<typeof stockAdjustmentSchema>;

interface StockAdjustmentModalProps {
  isOpen: boolean;
  onClose: () => void;
  product: Product | null;
  adjustmentType?: 'ADD' | 'REMOVE' | 'ADJUST';
}

export function StockAdjustmentModal({
  isOpen,
  onClose,
  product,
  adjustmentType = 'ADD',
}: StockAdjustmentModalProps) {
  const queryClient = useQueryClient();

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<StockAdjustmentFormData>({
    resolver: zodResolver(stockAdjustmentSchema),
    defaultValues: {
      type: adjustmentType,
      quantity: 1,
      reason: '',
    },
  });

  const type = watch('type');

  // Add stock mutation
  const addStockMutation = useMutation({
    mutationFn: ({
      productId,
      quantity,
      reason,
    }: {
      productId: string;
      quantity: number;
      reason: string;
    }) => inventoryApi.addStock(productId, quantity, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['products-inventory'] });
      queryClient.invalidateQueries({ queryKey: ['low-stock'] });
      toast.success('Stock added successfully');
      reset();
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Remove stock mutation
  const removeStockMutation = useMutation({
    mutationFn: ({
      productId,
      quantity,
      reason,
    }: {
      productId: string;
      quantity: number;
      reason: string;
    }) => inventoryApi.removeStock(productId, quantity, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['products-inventory'] });
      queryClient.invalidateQueries({ queryKey: ['low-stock'] });
      toast.success('Stock removed successfully');
      reset();
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  // Adjust stock mutation
  const adjustStockMutation = useMutation({
    mutationFn: ({
      productId,
      quantity,
      reason,
    }: {
      productId: string;
      quantity: number;
      reason: string;
    }) => inventoryApi.adjustStock(productId, quantity, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['products-inventory'] });
      queryClient.invalidateQueries({ queryKey: ['low-stock'] });
      toast.success('Stock adjusted successfully');
      reset();
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: StockAdjustmentFormData) => {
    if (!product) return;

    const params = { productId: product.id, quantity: data.quantity, reason: data.reason };

    switch (data.type) {
      case 'ADD':
        addStockMutation.mutate(params);
        break;
      case 'REMOVE':
        removeStockMutation.mutate(params);
        break;
      case 'ADJUST':
        adjustStockMutation.mutate(params);
        break;
    }
  };

  const isLoading =
    addStockMutation.isPending || removeStockMutation.isPending || adjustStockMutation.isPending;

  const getTitle = () => {
    switch (type) {
      case 'ADD':
        return 'Add Stock';
      case 'REMOVE':
        return 'Remove Stock';
      case 'ADJUST':
        return 'Adjust Stock';
    }
  };

  const getIcon = () => {
    switch (type) {
      case 'ADD':
        return <ArrowUpCircle className="w-5 h-5 text-green-600" />;
      case 'REMOVE':
        return <ArrowDownCircle className="w-5 h-5 text-red-600" />;
      case 'ADJUST':
        return <ArrowUpCircle className="w-5 h-5 text-blue-600" />;
    }
  };

  if (!product) return null;

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={getTitle()} size="md">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        {/* Product Info */}
        <div className="bg-gray-50 rounded-lg p-4">
          <div className="flex items-center gap-3">
            {getIcon()}
            <div>
              <p className="font-medium text-gray-900">{product.name}</p>
              <p className="text-sm text-gray-500">SKU: {product.sku}</p>
            </div>
          </div>
          <div className="mt-3 grid grid-cols-3 gap-4 text-sm">
            <div>
              <p className="text-gray-500">Current Stock</p>
              <p className="font-medium">{product.quantity}</p>
            </div>
            <div>
              <p className="text-gray-500">Available</p>
              <p className="font-medium">{product.available_qty}</p>
            </div>
            <div>
              <p className="text-gray-500">Reserved</p>
              <p className="font-medium">{product.reserved_qty}</p>
            </div>
          </div>
        </div>

        <Select
          label="Adjustment Type"
          options={[
            { value: 'ADD', label: 'Add Stock (Restock)' },
            { value: 'REMOVE', label: 'Remove Stock (Damaged/Lost)' },
            { value: 'ADJUST', label: 'Set Exact Quantity' },
          ]}
          error={errors.type?.message}
          required
          {...register('type')}
        />

        <Input
          label={type === 'ADJUST' ? 'New Quantity' : 'Quantity'}
          type="number"
          min="1"
          placeholder="Enter quantity"
          error={errors.quantity?.message}
          required
          {...register('quantity', { valueAsNumber: true })}
        />

        <div>
          <label className="label">
            Reason <span className="text-red-500">*</span>
          </label>
          <textarea
            className="input min-h-[80px]"
            placeholder={
              type === 'ADD'
                ? 'e.g., Received shipment from supplier...'
                : type === 'REMOVE'
                  ? 'e.g., Damaged during handling...'
                  : 'e.g., Physical count adjustment...'
            }
            {...register('reason')}
          />
          {errors.reason && <p className="text-sm text-red-600 mt-1">{errors.reason.message}</p>}
        </div>

        <div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
          <Button variant="secondary" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button
            type="submit"
            loading={isLoading}
            variant={type === 'REMOVE' ? 'danger' : 'primary'}
          >
            {type === 'ADD' ? 'Add Stock' : type === 'REMOVE' ? 'Remove Stock' : 'Update Stock'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
