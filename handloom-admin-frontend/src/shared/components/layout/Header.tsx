import { Menu, Transition } from '@headlessui/react';
import { clsx } from 'clsx';
import { Bell, LogOut, Menu as MenuIcon, Settings, User } from 'lucide-react';
import { Fragment } from 'react';
import { useNavigate } from 'react-router-dom';

import { authApi } from '@/features/auth/api';
import { useAuthStore } from '@/shared/stores/authStore';
import { useUIStore } from '@/shared/stores/uiStore';

export function Header() {
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const sidebarCollapsed = useUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useUIStore((s) => s.toggleSidebar);

  const handleLogout = async () => {
    try {
      await authApi.logout();
    } catch {
      // Server logout failed, still clear local state
    }
    logout();
    navigate('/login');
  };

  return (
    <header
      className={clsx(
        'fixed top-0 right-0 z-30 h-16 bg-white/95 backdrop-blur-sm border-b border-gray-100 shadow-sm transition-all duration-300',
        sidebarCollapsed ? 'left-20' : 'left-64'
      )}
    >
      <div className="flex items-center justify-between h-full px-6">
        {/* Left side */}
        <div className="flex items-center gap-4">
          <button
            onClick={toggleSidebar}
            className="p-2 rounded-xl text-gray-500 hover:bg-gray-100 hover:text-gray-700 transition-all duration-200 lg:hidden"
          >
            <MenuIcon className="w-5 h-5" />
          </button>
        </div>

        {/* Right side */}
        <div className="flex items-center gap-3">
          {/* Notifications */}
          <button
            onClick={() => navigate('/notifications')}
            className="relative p-2.5 rounded-xl text-gray-500 hover:bg-gray-100 hover:text-gray-700 transition-all duration-200"
          >
            <Bell className="w-5 h-5" />
          </button>

          {/* User menu */}
          <Menu as="div" className="relative">
            <Menu.Button className="flex items-center gap-3 p-1.5 rounded-xl hover:bg-gray-100 transition-all duration-200">
              <div className="w-9 h-9 bg-gradient-to-br from-primary-100 to-orange-100 rounded-xl flex items-center justify-center ring-1 ring-inset ring-primary-200/50">
                <User className="w-4 h-4 text-primary-600" />
              </div>
              <div className="hidden sm:block text-left">
                <p className="text-sm font-semibold text-gray-900">
                  {user?.first_name || user?.email?.split('@')[0] || 'Admin'}
                </p>
                <p className="text-xs text-gray-500 capitalize">{user?.role?.toLowerCase()}</p>
              </div>
            </Menu.Button>

            <Transition
              as={Fragment}
              enter="transition ease-out duration-200"
              enterFrom="transform opacity-0 scale-95 -translate-y-1"
              enterTo="transform opacity-100 scale-100 translate-y-0"
              leave="transition ease-in duration-150"
              leaveFrom="transform opacity-100 scale-100 translate-y-0"
              leaveTo="transform opacity-0 scale-95 -translate-y-1"
            >
              <Menu.Items className="absolute right-0 mt-2 w-56 origin-top-right bg-white rounded-2xl shadow-lg shadow-gray-200/50 ring-1 ring-gray-100 focus:outline-none p-1.5">
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => navigate('/settings/profile')}
                      className={clsx(
                        'flex items-center gap-3 w-full px-3 py-2.5 text-sm font-medium rounded-xl transition-colors duration-150',
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      )}
                    >
                      <User className="w-4 h-4" />
                      Profile
                    </button>
                  )}
                </Menu.Item>
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={() => navigate('/settings')}
                      className={clsx(
                        'flex items-center gap-3 w-full px-3 py-2.5 text-sm font-medium rounded-xl transition-colors duration-150',
                        active ? 'bg-gray-100 text-gray-900' : 'text-gray-700'
                      )}
                    >
                      <Settings className="w-4 h-4" />
                      Settings
                    </button>
                  )}
                </Menu.Item>
                <div className="my-1.5 border-t border-gray-100" />
                <Menu.Item>
                  {({ active }) => (
                    <button
                      onClick={handleLogout}
                      className={clsx(
                        'flex items-center gap-3 w-full px-3 py-2.5 text-sm font-medium text-red-600 rounded-xl transition-colors duration-150',
                        active ? 'bg-red-50' : ''
                      )}
                    >
                      <LogOut className="w-4 h-4" />
                      Logout
                    </button>
                  )}
                </Menu.Item>
              </Menu.Items>
            </Transition>
          </Menu>
        </div>
      </div>
    </header>
  );
}
