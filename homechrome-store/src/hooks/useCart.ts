'use client';

import { useCallback, useEffect, useRef } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { notifications } from '@mantine/notifications';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { useAuthStore } from '@/stores/auth';
import { useCartStore } from '@/stores/cart';
import { CartWithItems } from '@/types';

const CART_QUERY_KEY = ['cart'] as const;

export function useCart() {
  const queryClient = useQueryClient();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const setCartStore = useCartStore((s) => s.setCart);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: CART_QUERY_KEY,
    queryFn: async (): Promise<CartWithItems> => {
      const { data } = await api.get<CartWithItems>(ROUTES.CART.ROOT);
      return data;
    },
    staleTime: 5 * 60 * 1000,
    retry: 1,
  });

  const cart = data ?? null;
  const loading = isLoading;

  // Sync Zustand store for Header badge
  useEffect(() => {
    setCartStore(cart?.items ?? []);
  }, [cart, setCartStore]);

  // Invalidate cart on any auth transition (login merges guest→customer; logout needs fresh state)
  const wasAuthenticated = useRef(isAuthenticated);
  useEffect(() => {
    if (wasAuthenticated.current !== isAuthenticated) {
      queryClient.invalidateQueries({ queryKey: CART_QUERY_KEY });
    }
    wasAuthenticated.current = isAuthenticated;
  }, [isAuthenticated, queryClient]);

  const updateCache = useCallback(
    (newData: CartWithItems | null) => {
      queryClient.setQueryData<CartWithItems | null>(CART_QUERY_KEY, newData);
      // Sync Zustand immediately instead of waiting for useEffect
      setCartStore(newData?.items ?? []);
    },
    [queryClient, setCartStore],
  );

  const addItemMutation = useMutation({
    mutationFn: async ({
      productId,
      quantity,
    }: {
      productId: string;
      quantity: number;
    }) => {
      const { data } = await api.post<CartWithItems>(
        ROUTES.CART.ITEMS,
        { product_id: productId, quantity },
      );
      return data;
    },
    onSuccess: (data) => {
      updateCache(data);
      notifications.show({ message: 'Added to cart', color: 'teal' });
    },
    onError: () => {
      notifications.show({ message: 'Failed to add to cart', color: 'red' });
    },
  });

  const updateQuantityMutation = useMutation({
    mutationFn: async ({
      productId,
      quantity,
    }: {
      productId: string;
      quantity: number;
    }) => {
      const { data } = await api.patch<CartWithItems>(
        ROUTES.CART.ITEM(productId),
        { quantity },
      );
      return data;
    },
    onSuccess: updateCache,
    onError: () => {
      notifications.show({ message: 'Failed to update cart', color: 'red' });
    },
  });

  const removeItemMutation = useMutation({
    mutationFn: async (productId: string) => {
      const { data } = await api.delete<CartWithItems>(
        ROUTES.CART.ITEM(productId),
      );
      return data;
    },
    onSuccess: (data) => {
      updateCache(data);
      notifications.show({ message: 'Item removed from cart' });
    },
    onError: () => {
      notifications.show({ message: 'Failed to remove item', color: 'red' });
    },
  });

  const clearCartMutation = useMutation({
    mutationFn: async () => {
      await api.delete(ROUTES.CART.ROOT);
    },
    onSuccess: () => {
      updateCache(null);
      notifications.show({ message: 'Cart cleared' });
    },
    onError: () => {
      notifications.show({ message: 'Failed to clear cart', color: 'red' });
    },
  });

  const addItem = async (productId: string, quantity: number) =>
    addItemMutation.mutateAsync({ productId, quantity });

  const updateQuantity = async (productId: string, quantity: number) =>
    updateQuantityMutation.mutateAsync({ productId, quantity });

  const removeItem = async (productId: string) =>
    removeItemMutation.mutateAsync(productId);

  const clearCart = async () => clearCartMutation.mutateAsync();

  return {
    cart,
    loading,
    error,
    fetchCart: refetch,
    addItem,
    updateQuantity,
    removeItem,
    clearCart,
  };
}
