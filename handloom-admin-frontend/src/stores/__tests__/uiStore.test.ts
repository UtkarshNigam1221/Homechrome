import { beforeEach, describe, expect, it } from 'vitest';

import { useUIStore } from '../uiStore';

describe('uiStore', () => {
  beforeEach(() => {
    useUIStore.setState({
      sidebarOpen: true,
      sidebarCollapsed: false,
      theme: 'light',
    });
  });

  it('toggles sidebar collapsed state', () => {
    useUIStore.getState().toggleSidebarCollapse();
    expect(useUIStore.getState().sidebarCollapsed).toBe(true);
    useUIStore.getState().toggleSidebarCollapse();
    expect(useUIStore.getState().sidebarCollapsed).toBe(false);
  });

  it('toggles sidebar open state', () => {
    useUIStore.getState().toggleSidebar();
    expect(useUIStore.getState().sidebarOpen).toBe(false);
  });

  it('sets theme', () => {
    useUIStore.getState().setTheme('dark');
    expect(useUIStore.getState().theme).toBe('dark');
  });
});
