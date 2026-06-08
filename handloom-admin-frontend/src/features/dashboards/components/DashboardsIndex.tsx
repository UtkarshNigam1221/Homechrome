import { Link } from 'react-router-dom';

// Landing tab: quick-link cards to each sub-dashboard.

const CARDS = [
  {
    to: '/dashboards/funnel',
    title: 'Funnel',
    description: 'Session-to-payment conversion, stage KPIs, device split, UTM acquisition.',
    accent: 'from-indigo-50 to-white',
  },
  {
    to: '/dashboards/products',
    title: 'Product analytics',
    description:
      'Top viewed/purchased products, search behavior, coupon performance, filter usage.',
    accent: 'from-emerald-50 to-white',
  },
  {
    to: '/dashboards/geography',
    title: 'Geography',
    description: 'India geomap of orders + revenue by city, state distribution.',
    accent: 'from-amber-50 to-white',
  },
  {
    to: '/dashboards/rum',
    title: 'RUM & service health',
    description:
      'Web vitals, JS errors, HTTP errors, lambda cold starts, gateway calls, DB latency.',
    accent: 'from-rose-50 to-white',
  },
];

export function DashboardsIndex() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-neutral-900">Dashboards</h1>
        <p className="text-sm text-neutral-600">
          Pick a dashboard. All views share the time range above.
        </p>
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {CARDS.map((card) => (
          <Link
            key={card.to}
            to={card.to}
            className={`block rounded-lg border border-neutral-200 bg-gradient-to-br ${card.accent} p-5 shadow-sm transition hover:border-indigo-300 hover:shadow-md`}
          >
            <div className="text-base font-semibold text-neutral-900">{card.title}</div>
            <p className="mt-1 text-sm text-neutral-600">{card.description}</p>
            <div className="mt-3 text-xs font-medium text-indigo-600">Open →</div>
          </Link>
        ))}
      </div>
    </div>
  );
}
