import { clsx } from 'clsx';
import {
  BarChart3,
  Bell,
  ChevronLeft,
  ChevronRight,
  FileText,
  FolderTree,
  LayoutDashboard,
  Package,
  Palette,
  Percent,
  Settings,
  ShoppingCart,
  Tag,
  Upload,
  UserCircle,
  Users,
  Warehouse,
} from 'lucide-react';
import { NavLink, useLocation } from 'react-router-dom';

import { useAuthStore } from '../../stores/authStore';
import { useUIStore } from '../../stores/uiStore';

const navigation = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard },
  { name: 'Categories', href: '/categories', icon: FolderTree },
  { name: 'Designs', href: '/designs', icon: Palette },
  { name: 'Products', href: '/products', icon: Package },
  { name: 'Inventory', href: '/inventory', icon: Warehouse },
  { name: 'Orders', href: '/orders', icon: ShoppingCart },
  { name: 'Customers', href: '/customers', icon: Users },
  { name: 'Artisans', href: '/artisans', icon: UserCircle },
  { name: 'Pricing Rules', href: '/pricing', icon: Tag },
  { name: 'Coupons', href: '/coupons', icon: Percent },
  { name: 'Analytics', href: '/analytics', icon: BarChart3 },
  { name: 'Reports', href: '/reports', icon: FileText },
  { name: 'Bulk Operations', href: '/bulk', icon: Upload },
  { name: 'Notifications', href: '/notifications', icon: Bell },
  { name: 'Settings', href: '/settings', icon: Settings },
];

const adminNavigation = [{ name: 'Users', href: '/users', icon: Users }];

export function Sidebar() {
  const location = useLocation();
  const { sidebarCollapsed, toggleSidebarCollapse } = useUIStore();
  const { user } = useAuthStore();

  const isAdmin = user?.role === 'ADMIN';

  return (
    <aside
      className={clsx(
        'fixed left-0 top-0 z-40 h-screen bg-white/95 backdrop-blur-sm border-r border-gray-100 transition-all duration-300 shadow-sm',
        sidebarCollapsed ? 'w-20' : 'w-64'
      )}
    >
      {/* Logo */}
      <div className="flex items-center justify-between h-16 px-4 border-b border-gray-100">
        {!sidebarCollapsed && (
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 bg-gradient-to-br from-primary-500 to-primary-600 rounded-xl flex items-center justify-center shadow-sm shadow-primary-500/30">
              <span className="text-white font-bold text-sm">H</span>
            </div>
            <span className="font-semibold text-gray-900 tracking-tight">Handloom Admin</span>
          </div>
        )}
        {sidebarCollapsed && (
          <div className="w-9 h-9 bg-gradient-to-br from-primary-500 to-primary-600 rounded-xl flex items-center justify-center mx-auto shadow-sm shadow-primary-500/30">
            <span className="text-white font-bold text-sm">H</span>
          </div>
        )}
        <button
          onClick={toggleSidebarCollapse}
          className={clsx(
            'p-1.5 rounded-lg text-gray-500 hover:bg-gray-100 transition-all duration-200',
            sidebarCollapsed &&
              'absolute -right-3 top-6 bg-white border border-gray-200 shadow-md hover:shadow-lg'
          )}
        >
          {sidebarCollapsed ? (
            <ChevronRight className="w-4 h-4" />
          ) : (
            <ChevronLeft className="w-4 h-4" />
          )}
        </button>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 space-y-1 overflow-y-auto h-[calc(100vh-64px)]">
        <div className="space-y-1">
          {navigation.map((item) => {
            const isActive =
              location.pathname === item.href ||
              (item.href !== '/' && location.pathname.startsWith(item.href));

            return (
              <NavLink
                key={item.name}
                to={item.href}
                className={clsx(
                  'sidebar-link',
                  isActive ? 'sidebar-link-active' : 'sidebar-link-inactive',
                  sidebarCollapsed && 'justify-center px-2'
                )}
                title={sidebarCollapsed ? item.name : undefined}
              >
                <item.icon className="w-5 h-5 flex-shrink-0" />
                {!sidebarCollapsed && <span>{item.name}</span>}
              </NavLink>
            );
          })}
        </div>

        {isAdmin && (
          <>
            <div className="pt-4 mt-4 border-t border-gray-200">
              {!sidebarCollapsed && (
                <p className="px-3 mb-2 text-xs font-semibold text-gray-400 uppercase tracking-wider">
                  Administration
                </p>
              )}
              <div className="space-y-1">
                {adminNavigation.map((item) => {
                  const isActive =
                    location.pathname === item.href || location.pathname.startsWith(item.href);

                  return (
                    <NavLink
                      key={item.name}
                      to={item.href}
                      className={clsx(
                        'sidebar-link',
                        isActive ? 'sidebar-link-active' : 'sidebar-link-inactive',
                        sidebarCollapsed && 'justify-center px-2'
                      )}
                      title={sidebarCollapsed ? item.name : undefined}
                    >
                      <item.icon className="w-5 h-5 flex-shrink-0" />
                      {!sidebarCollapsed && <span>{item.name}</span>}
                    </NavLink>
                  );
                })}
              </div>
            </div>
          </>
        )}
      </nav>
    </aside>
  );
}
