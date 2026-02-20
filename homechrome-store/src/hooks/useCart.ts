'use client';

import { useCallback, useEffect, useState } from 'react';

import api from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { CartWithItems } from '@/types';

export function useCart() {
  const [cart, setCart] = useState<CartWithItems | null>(null);
  const [loading, setLoading] = useState(false);
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);

  const fetchCart = useCallback(async () => {
    if (!isAuthenticated) return;
    setLoading(true);
    try {
      const { data } = await api.get<CartWithItems>('/api/v1/store/cart');
      setCart(data);
    } catch {
      /* ignore */
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated]);

  useEffect(() => {
    fetchCart();
  }, [fetchCart]);

  const addItem = async (productId: string, quantity: number) => {
    const { data } = await api.post<CartWithItems>('/api/v1/store/cart/items', {
      product_id: productId,
      quantity,
    });
    setCart(data);
    return data;
  };

  const updateQuantity = async (productId: string, quantity: number) => {
    const { data } = await api.patch<CartWithItems>(
      `/api/v1/store/cart/items/${productId}`,
      { quantity },
    );
    setCart(data);
    return data;
  };

  const removeItem = async (productId: string) => {
    const { data } = await api.delete<CartWithItems>(
      `/api/v1/store/cart/items/${productId}`,
    );
    setCart(data);
    return data;
  };

  const clearCart = async () => {
    await api.delete('/api/v1/store/cart');
    setCart(null);
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
