'use client';

import { useCallback, useEffect, useRef } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { toast } from 'sonner';

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

  const { data, isLoading, refetch } = useQuery({
    queryKey: CART_QUERY_KEY,
    queryFn: async (): Promise<CartWithItems> => {
      const { data } = await api.get<CartWithItems>(ROUTES.CART.ROOT);
      return data;
    },
    enabled: isAuthenticated,
    staleTime: 5 * 60 * 1000,
  });

  const cart = data ?? null;
  const loading = isLoading;

  // Sync Zustand store for Header badge
  useEffect(() => {
    setCartStore(cart?.items ?? []);
  }, [cart, setCartStore]);

  // Clear cache on logout (authenticated → unauthenticated transition)
  const wasAuthenticated = useRef(isAuthenticated);
  useEffect(() => {
    if (wasAuthenticated.current && !isAuthenticated) {
      queryClient.removeQueries({ queryKey: CART_QUERY_KEY });
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
      toast.success('Added to cart');
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
      toast('Item removed from cart');
    },
  });

  const clearCartMutation = useMutation({
    mutationFn: async () => {
      await api.delete(ROUTES.CART.ROOT);
    },
    onSuccess: () => {
      updateCache(null);
      toast('Cart cleared');
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
    fetchCart: refetch,
    addItem,
    updateQuantity,
    removeItem,
    clearCart,
  };
}
