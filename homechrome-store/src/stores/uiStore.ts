import { create } from 'zustand';

interface UIState {
  miniCartOpen: boolean;
  openMiniCart: () => void;
  closeMiniCart: () => void;
  toggleMiniCart: () => void;
  notificationsOpen: boolean;
  openNotifications: () => void;
  closeNotifications: () => void;
}

export const useUIStore = create<UIState>((set) => ({
  miniCartOpen: false,
  openMiniCart: () => set({ miniCartOpen: true }),
  closeMiniCart: () => set({ miniCartOpen: false }),
  toggleMiniCart: () => set((s) => ({ miniCartOpen: !s.miniCartOpen })),
  notificationsOpen: false,
  openNotifications: () => set({ notificationsOpen: true }),
  closeNotifications: () => set({ notificationsOpen: false }),
}));
