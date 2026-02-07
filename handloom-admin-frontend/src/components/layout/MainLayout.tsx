import { clsx } from 'clsx';
import { Outlet } from 'react-router-dom';

import { useUIStore } from '../../stores/uiStore';
import { Header } from './Header';
import { Sidebar } from './Sidebar';

export function MainLayout() {
  const { sidebarCollapsed } = useUIStore();

  return (
    <div className="min-h-screen bg-gray-50">
      <Sidebar />
      <Header />
      <main
        className={clsx(
          'pt-16 min-h-screen transition-all duration-300',
          sidebarCollapsed ? 'ml-20' : 'ml-64'
        )}
      >
        <div className="p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
