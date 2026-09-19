import type { PushBroadcastStatus } from '../types';

/** Shared so the history row and its detail modal cannot drift apart. */
export const broadcastStatusVariant: Record<PushBroadcastStatus, 'success' | 'warning' | 'danger'> =
  {
    SUCCESS: 'success',
    PARTIAL: 'warning',
    FAILED: 'danger',
  };
