import { useState } from 'react';

import { useCart } from '@/hooks/useCart';
import { track } from '@/lib/analytics';
import { useCartStore } from '@/stores/cart';
import { Product } from '@/types';

export function useProductCartActions(product: Product) {
  const [quantity, setQuantity] = useState(1);
  const [loading, setLoading] = useState(false);
  const { addItem, updateQuantity, removeItem } = useCart();
  const cartQty = useCartStore((s) => s.getQuantity(product.id));

  const handleAdd = async () => {
    setLoading(true);
    try {
      await addItem(product.id, quantity);
      track('add_to_cart', {
        product_id: product.id,
        product_name: product.name,
        category_id: product.category_id,
        price: product.selling_price,
        quantity,
      });
    } catch {
      /* useCart shows error toast */
    } finally {
      setLoading(false);
    }
  };

  const handleIncrement = async () => {
    setLoading(true);
    try {
      await updateQuantity(product.id, cartQty + 1);
    } catch {
      /* useCart shows error toast */
    } finally {
      setLoading(false);
    }
  };

  const handleDecrement = async () => {
    setLoading(true);
    try {
      if (cartQty <= 1) {
        await removeItem(product.id);
      } else {
        await updateQuantity(product.id, cartQty - 1);
      }
    } catch {
      /* useCart shows error toast */
    } finally {
      setLoading(false);
    }
  };

  const incrementQuantity = () => setQuantity((q) => q + 1);
  const decrementQuantity = () => setQuantity((q) => Math.max(1, q - 1));

  return {
    quantity,
    cartQty,
    loading,
    incrementQuantity,
    decrementQuantity,
    handleAdd,
    handleIncrement,
    handleDecrement,
  };
}
