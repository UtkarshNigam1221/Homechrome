import { beforeEach, describe, expect, it } from 'vitest';

import { useAuthStore } from '../authStore';

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      isAuthenticated: false,
      isLoading: true,
    });
  });

  it('starts unauthenticated', () => {
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.user).toBeNull();
  });

  it('login sets user and isAuthenticated', () => {
    const user = { id: '1', email: 'test@test.com', name: 'Test', role: 'ADMIN' };
    useAuthStore.getState().login(user as never);
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(true);
    expect(state.user?.email).toBe('test@test.com');
    expect(state.isLoading).toBe(false);
  });

  it('logout clears user', () => {
    useAuthStore.getState().login({ id: '1', email: 'test@test.com', name: 'Test', role: 'ADMIN' } as never);
    useAuthStore.getState().logout();
    const state = useAuthStore.getState();
    expect(state.isAuthenticated).toBe(false);
    expect(state.user).toBeNull();
  });
});
