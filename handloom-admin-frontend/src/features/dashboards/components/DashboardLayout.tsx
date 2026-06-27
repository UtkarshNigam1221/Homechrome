import { NavLink, Outlet } from 'react-router-dom';

import { DASHBOARD_TABS } from '../constants';
import { NeonAuthGate } from './NeonAuthGate';
import { TimeRangePicker } from './TimeRangePicker';

export function DashboardLayout() {
  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-neutral-200 bg-white px-6 py-3">
        <nav className="flex gap-1">
          {DASHBOARD_TABS.map((tab) => (
            <NavLink
              key={tab.to}
              to={tab.to}
              end={tab.end}
              className={({ isActive }) =>
                'rounded-md px-3 py-1.5 text-sm font-medium transition-colors ' +
                (isActive
                  ? 'bg-indigo-50 text-indigo-700'
                  : 'text-neutral-600 hover:bg-neutral-100')
              }
            >
              {tab.label}
            </NavLink>
          ))}
        </nav>
        <TimeRangePicker />
      </div>
      <div className="flex-1 overflow-auto bg-neutral-50">
        <NeonAuthGate>
          <div className="p-6">
            <Outlet />
          </div>
        </NeonAuthGate>
      </div>
    </div>
  );
}
