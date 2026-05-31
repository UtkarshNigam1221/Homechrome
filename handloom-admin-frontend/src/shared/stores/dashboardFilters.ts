import { useMemo } from 'react';

import { create } from 'zustand';

// Time-range presets the dashboards offer. `custom` opens a date picker.
type Range = '1h' | '6h' | '24h' | '7d' | '30d' | 'custom';

interface DashboardFiltersState {
  range: Range;
  customFrom?: Date;
  customTo?: Date;
  setRange: (r: Range) => void;
  setCustom: (from: Date, to: Date) => void;
}

export const useDashboardFilters = create<DashboardFiltersState>((set) => ({
  range: '24h',
  setRange: (range) => set({ range }),
  setCustom: (customFrom, customTo) => set({ range: 'custom', customFrom, customTo }),
}));

function rangeToMs(range: Range): number {
  switch (range) {
    case '1h':
      return 1 * 60 * 60 * 1000;
    case '6h':
      return 6 * 60 * 60 * 1000;
    case '24h':
      return 24 * 60 * 60 * 1000;
    case '7d':
      return 7 * 24 * 60 * 60 * 1000;
    case '30d':
      return 30 * 24 * 60 * 60 * 1000;
    default:
      return 24 * 60 * 60 * 1000;
  }
}

// useResolvedRange returns memoised {from, to} for the current range.
// `from` and `to` reset to a stable bucket aligned to the START of the
// current minute, so calling this hook every render does NOT produce a
// new Date object each tick (which would cause infinite useQuery refetches).
export function useResolvedRange(): { from: Date; to: Date } {
  const range = useDashboardFilters((s) => s.range);
  const customFromMs = useDashboardFilters((s) => s.customFrom?.getTime());
  const customToMs = useDashboardFilters((s) => s.customTo?.getTime());

  // Round to the current minute → stable reference within a 60s window.
  // Window advances every minute, which is acceptable refresh cadence.
  const minuteBucket = useMemo(() => Math.floor(Date.now() / 60_000) * 60_000, []);

  return useMemo(() => {
    if (range === 'custom' && customFromMs && customToMs) {
      return { from: new Date(customFromMs), to: new Date(customToMs) };
    }
    const to = new Date(minuteBucket);
    const from = new Date(minuteBucket - rangeToMs(range));
    return { from, to };
  }, [range, customFromMs, customToMs, minuteBucket]);
}
