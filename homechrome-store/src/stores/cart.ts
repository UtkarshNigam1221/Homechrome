import { create } from 'zustand';

import { CartItem } from '@/types';

interface CartState {
  itemCount: number;
  items: CartItem[];
  setCart: (items: CartItem[]) => void;
  getQuantity: (productId: string) => number;
}

export const useCartStore = create<CartState>((set, get) => ({
  itemCount: 0,
  items: [],
  setCart: (items) =>
    set({
      items,
      itemCount: items.reduce((sum, i) => sum + i.quantity, 0),
    }),
  getQuantity: (productId) => {
    const item = get().items.find((i) => i.product_id === productId);
    return item?.quantity ?? 0;
  },
}));
