'use client';

import { useCallback, useEffect, useState } from 'react';

import api from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { useCartStore } from '@/stores/cart';
import { CartWithItems } from '@/types';

export function useCart() {
  const [cart, setCart] = useState<CartWithItems | null>(null);
  const [loading, setLoading] = useState(false);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const setCartStore = useCartStore((s) => s.setCart);

  const updateCart = useCallback(
    (data: CartWithItems | null) => {
      setCart(data);
      setCartStore(data?.items ?? []);
    },
    [setCartStore],
  );

  const fetchCart = useCallback(async () => {
    if (!isAuthenticated) return;
    setLoading(true);
    try {
      const { data } = await api.get<CartWithItems>('/api/v1/store/cart');
      updateCart(data);
    } catch {
      /* ignore */
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated, updateCart]);

  useEffect(() => {
    fetchCart();
  }, [fetchCart]);

  const addItem = async (productId: string, quantity: number) => {
    const { data } = await api.post<CartWithItems>('/api/v1/store/cart/items', {
      product_id: productId,
      quantity,
    });
    updateCart(data);
    return data;
  };

  const updateQuantity = async (productId: string, quantity: number) => {
    const { data } = await api.patch<CartWithItems>(
      `/api/v1/store/cart/items/${productId}`,
      { quantity },
    );
    updateCart(data);
    return data;
  };

  const removeItem = async (productId: string) => {
    const { data } = await api.delete<CartWithItems>(
      `/api/v1/store/cart/items/${productId}`,
    );
    updateCart(data);
    return data;
  };

  const clearCart = async () => {
    await api.delete('/api/v1/store/cart');
    updateCart(null);
  };

  return {
    cart,
    loading,
    fetchCart,
    addItem,
    updateQuantity,
    removeItem,
    clearCart,
  };
}
