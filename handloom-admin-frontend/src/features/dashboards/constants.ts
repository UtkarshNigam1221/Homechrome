// Single source of truth for dashboard routes + nav metadata.
// Reused by DashboardLayout (tab strip) and DashboardsIndex (link cards).

export const DASHBOARD_PATHS = {
  overview: '/dashboards',
  funnel: '/dashboards/funnel',
  products: '/dashboards/products',
  geography: '/dashboards/geography',
  rum: '/dashboards/rum',
} as const;

export const DASHBOARD_TABS = [
  { to: DASHBOARD_PATHS.overview, label: 'Overview', end: true },
  { to: DASHBOARD_PATHS.funnel, label: 'Funnel' },
  { to: DASHBOARD_PATHS.products, label: 'Products' },
  { to: DASHBOARD_PATHS.geography, label: 'Geography' },
  { to: DASHBOARD_PATHS.rum, label: 'RUM & Service Health' },
];

// Device-type → swatch. Shared by funnel + product device splits.
export const DEVICE_COLORS: Record<string, string> = {
  mobile: '#6366f1', // indigo
  desktop: '#06b6d4', // cyan
  tablet: '#f59e0b', // amber
  unknown: '#a3a3a3', // neutral
};

export const DASHBOARD_CARDS = [
  {
    to: DASHBOARD_PATHS.funnel,
    title: 'Funnel',
    description: 'Session-to-payment conversion, stage KPIs, device split, UTM acquisition.',
    accent: 'from-indigo-50 to-white',
  },
  {
    to: DASHBOARD_PATHS.products,
    title: 'Product analytics',
    description:
      'Top viewed/purchased products, search behavior, coupon performance, filter usage.',
    accent: 'from-emerald-50 to-white',
  },
  {
    to: DASHBOARD_PATHS.geography,
    title: 'Geography',
    description: 'India geomap of orders + revenue by city, state distribution.',
    accent: 'from-amber-50 to-white',
  },
  {
    to: DASHBOARD_PATHS.rum,
    title: 'RUM & service health',
    description:
      'Web vitals, JS errors, HTTP errors, lambda cold starts, gateway calls, DB latency.',
    accent: 'from-rose-50 to-white',
  },
];
