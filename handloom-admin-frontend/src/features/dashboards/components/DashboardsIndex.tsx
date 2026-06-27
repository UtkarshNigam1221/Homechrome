import { Link } from 'react-router-dom';

import { DASHBOARD_CARDS } from '../constants';

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
        {DASHBOARD_CARDS.map((card) => (
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
