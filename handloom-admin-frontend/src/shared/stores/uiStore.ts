import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface NotificationPrefs {
  orderUpdates: boolean;
  inventoryAlerts: boolean;
  systemNotifications: boolean;
  emailNotifications: boolean;
}

interface UIState {
  sidebarOpen: boolean;
  sidebarCollapsed: boolean;
  theme: 'light' | 'dark';
  notificationPrefs: NotificationPrefs;

  // Actions
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  toggleSidebarCollapse: () => void;
  setTheme: (theme: 'light' | 'dark') => void;
  setNotificationPref: (key: keyof NotificationPrefs, value: boolean) => void;
}

export const useUIStore = create<UIState>()(
  persist(
    (set) => ({
      sidebarOpen: true,
      sidebarCollapsed: false,
      theme: 'light',
      notificationPrefs: {
        orderUpdates: true,
        inventoryAlerts: true,
        systemNotifications: true,
        emailNotifications: false,
      },

      toggleSidebar: () => set((state) => ({ sidebarOpen: !state.sidebarOpen })),
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      toggleSidebarCollapse: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      setTheme: (theme) => set({ theme }),
      setNotificationPref: (key, value) =>
        set((state) => ({
          notificationPrefs: { ...state.notificationPrefs, [key]: value },
        })),
    }),
    {
      name: 'handloom-ui',
    }
  )
);
