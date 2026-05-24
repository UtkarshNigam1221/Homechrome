import { create } from 'zustand';

interface UIState {
  miniCartOpen: boolean;
  openMiniCart: () => void;
  closeMiniCart: () => void;
  toggleMiniCart: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  miniCartOpen: false,
  openMiniCart: () => set({ miniCartOpen: true }),
  closeMiniCart: () => set({ miniCartOpen: false }),
  toggleMiniCart: () => set((s) => ({ miniCartOpen: !s.miniCartOpen })),
}));
