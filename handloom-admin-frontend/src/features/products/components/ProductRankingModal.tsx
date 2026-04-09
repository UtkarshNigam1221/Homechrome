import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { GripVertical } from 'lucide-react';
import { useMemo, useState } from 'react';
import toast from 'react-hot-toast';

import { getErrorMessage } from '@/shared/api/client';
import { Badge, Button, Modal } from '@/shared/components/ui';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency } from '@/shared/utils/currency';

import { productsApi } from '../api';
import type { Product } from '../types';

interface ProductRankingModalProps {
  isOpen: boolean;
  onClose: () => void;
  categoryId: string;
  categoryName: string;
}

function SortableProduct({ product }: { product: Product }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: product.id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-center gap-3 rounded-lg border bg-white p-3 ${isDragging ? 'shadow-lg' : ''}`}
    >
      <button
        type="button"
        className="cursor-grab touch-none text-gray-400 hover:text-gray-600"
        {...attributes}
        {...listeners}
      >
        <GripVertical className="h-5 w-5" />
      </button>

      {product.images?.[0] && (
        <img
          src={product.images[0].url}
          alt={product.name}
          className="h-10 w-10 rounded object-cover"
        />
      )}

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-gray-900">{product.name}</p>
        <p className="text-xs text-gray-500">{product.sku}</p>
      </div>

      <div className="text-right text-sm text-gray-600">
        {formatCurrency(product.selling_price)}
      </div>

      <Badge variant={getStatusBadgeVariant(product.status)}>{product.status}</Badge>
    </div>
  );
}

export function ProductRankingModal({
  isOpen,
  onClose,
  categoryId,
  categoryName,
}: ProductRankingModalProps) {
  const queryClient = useQueryClient();
  // Track custom ordering as an array of product IDs; null means use server order
  const [customOrder, setCustomOrder] = useState<string[] | null>(null);

  const { data: productsData, isLoading } = useQuery({
    queryKey: ['products-for-ranking', categoryId],
    queryFn: () => productsApi.list({ category_id: categoryId, limit: 100 }),
    enabled: isOpen && !!categoryId,
  });

  // Sort server data by sort_order, then name
  const sortedServerProducts = useMemo(() => {
    if (!productsData?.items) return [];
    return [...productsData.items].sort((a, b) => {
      if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order;
      return a.name.localeCompare(b.name);
    });
  }, [productsData]);

  // Derive the final displayed order: custom drag order if set, otherwise server order
  const orderedProducts = useMemo(() => {
    if (!customOrder) return sortedServerProducts;
    const productMap = new Map(sortedServerProducts.map((p) => [p.id, p]));
    return customOrder.map((id) => productMap.get(id)).filter(Boolean) as Product[];
  }, [customOrder, sortedServerProducts]);

  const reorderMutation = useMutation({
    mutationFn: () =>
      productsApi.reorder(
        categoryId,
        orderedProducts.map((p) => p.id)
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['products-for-ranking'] });
      toast.success('Product order saved');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const currentIds = orderedProducts.map((p) => p.id);
    const oldIndex = currentIds.indexOf(active.id as string);
    const newIndex = currentIds.indexOf(over.id as string);
    setCustomOrder(arrayMove(currentIds, oldIndex, newIndex));
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`Manage Order \u2014 ${categoryName}`}
      size="lg"
    >
      <div className="space-y-4">
        <p className="text-sm text-gray-500">
          Drag products to set their display order. The order here determines how products appear on
          the storefront.
        </p>

        {isLoading ? (
          <div className="py-8 text-center text-gray-500">Loading products...</div>
        ) : orderedProducts.length === 0 ? (
          <div className="py-8 text-center text-gray-500">No products in this category.</div>
        ) : (
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={orderedProducts.map((p) => p.id)}
              strategy={verticalListSortingStrategy}
            >
              <div className="max-h-[60vh] space-y-2 overflow-y-auto">
                {orderedProducts.map((product, index) => (
                  <div key={product.id} className="flex items-center gap-2">
                    <span className="w-6 text-right text-xs text-gray-400">{index + 1}</span>
                    <div className="flex-1">
                      <SortableProduct product={product} />
                    </div>
                  </div>
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}

        <div className="flex justify-end gap-3 border-t pt-4">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={() => reorderMutation.mutate()}
            loading={reorderMutation.isPending}
            disabled={orderedProducts.length === 0 || reorderMutation.isPending}
          >
            Save Order
          </Button>
        </div>
      </div>
    </Modal>
  );
}
