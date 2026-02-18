// Colors aligned with Tailwind v4 @theme in index.css
export const CHART_COLORS = {
  primary: '#f97316', // primary-500
  primaryLight: '#fb923c', // primary-400
  blue: '#3b82f6', // blue-500
  amber: '#f59e0b', // amber-500
  emerald: '#10b981', // emerald-500
  violet: '#8b5cf6', // violet-500
  grid: '#f0f0f0',
  axis: '#9ca3af', // gray-400
} as const;

export const PIE_COLORS = [
  CHART_COLORS.primary,
  CHART_COLORS.amber,
  CHART_COLORS.emerald,
  CHART_COLORS.blue,
  CHART_COLORS.violet,
];
