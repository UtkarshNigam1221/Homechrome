import { useDashboardFilters } from '@/shared/stores/dashboardFilters';

// Pill button group. Selected range = filled indigo, others = white hover-grey.
// `custom` range button comes later (N5) with a popover date picker.
const RANGES = [
  { id: '1h', label: '1h' },
  { id: '6h', label: '6h' },
  { id: '24h', label: '24h' },
  { id: '7d', label: '7d' },
  { id: '30d', label: '30d' },
] as const;

export function TimeRangePicker() {
  const range = useDashboardFilters((s) => s.range);
  const setRange = useDashboardFilters((s) => s.setRange);

  return (
    <div className="inline-flex rounded-md border border-neutral-200 bg-white shadow-sm">
      {RANGES.map((r) => (
        <button
          key={r.id}
          onClick={() => setRange(r.id)}
          className={
            'px-3 py-1.5 text-xs font-medium transition-colors first:rounded-l-md last:rounded-r-md ' +
            (range === r.id
              ? 'bg-indigo-600 text-white'
              : 'bg-white text-neutral-600 hover:bg-neutral-50')
          }
          type="button"
        >
          {r.label}
        </button>
      ))}
    </div>
  );
}
