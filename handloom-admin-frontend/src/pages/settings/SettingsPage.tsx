import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { clsx } from 'clsx';
import { Bell, Lock, Palette, User as UserIcon } from 'lucide-react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast';
import { z } from 'zod';

import { authApi, getErrorMessage, usersApi } from '../../api';
import { Button, Card, CardHeader, Input } from '../../components/common';
import { useAuthStore } from '../../stores/authStore';
import { useUIStore } from '../../stores/uiStore';
import type { User } from '../../types';

type SettingsTab = 'profile' | 'security' | 'notifications' | 'appearance';

const tabs: { id: SettingsTab; label: string; icon: React.ElementType }[] = [
  { id: 'profile', label: 'Profile', icon: UserIcon },
  { id: 'security', label: 'Security', icon: Lock },
  { id: 'notifications', label: 'Notifications', icon: Bell },
  { id: 'appearance', label: 'Appearance', icon: Palette },
];

export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('profile');
  const { user } = useAuthStore();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="page-title">Settings</h1>
        <p className="page-subtitle">Manage your account settings and preferences</p>
      </div>

      <div className="flex flex-col lg:flex-row gap-6">
        {/* Sidebar */}
        <div className="lg:w-64 flex-shrink-0">
          <Card padding="sm">
            <nav role="tablist" className="space-y-1">
              {tabs.map((tab) => (
                <button
                  key={tab.id}
                  role="tab"
                  aria-selected={activeTab === tab.id}
                  aria-controls={`tabpanel-${tab.id}`}
                  onClick={() => setActiveTab(tab.id)}
                  className={clsx(
                    'flex items-center gap-3 w-full px-3 py-2 text-sm font-medium rounded-lg transition-colors',
                    activeTab === tab.id
                      ? 'bg-primary-50 text-primary-700'
                      : 'text-gray-600 hover:bg-gray-100'
                  )}
                >
                  <tab.icon className="w-5 h-5" />
                  {tab.label}
                </button>
              ))}
            </nav>
          </Card>
        </div>

        {/* Content */}
        <div className="flex-1" role="tabpanel" id={`tabpanel-${activeTab}`}>
          {activeTab === 'profile' && <ProfileSettings user={user} />}
          {activeTab === 'security' && <SecuritySettings />}
          {activeTab === 'notifications' && <NotificationSettings />}
          {activeTab === 'appearance' && <AppearanceSettings />}
        </div>
      </div>
    </div>
  );
}

interface ProfileFormData {
  first_name: string;
  last_name: string;
  email: string;
  phone: string;
}

function ProfileSettings({ user }: { user: User | null }) {
  const { setUser } = useAuthStore();
  const {
    register,
    handleSubmit,
    formState: { isDirty },
  } = useForm<ProfileFormData>({
    defaultValues: {
      first_name: user?.first_name || '',
      last_name: user?.last_name || '',
      email: user?.email || '',
      phone: user?.phone || '',
    },
  });

  const updateMutation = useMutation({
    mutationFn: (data: ProfileFormData) => {
      if (!user?.id) throw new Error('User not found');
      return usersApi.update(user.id, {
        first_name: data.first_name,
        last_name: data.last_name,
        phone: data.phone || undefined,
      });
    },
    onSuccess: (updatedUser) => {
      setUser(updatedUser);
      toast.success('Profile updated successfully');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const onSubmit = (data: ProfileFormData) => {
    updateMutation.mutate(data);
  };

  return (
    <Card>
      <CardHeader title="Profile Information" subtitle="Update your personal information" />
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input label="First Name" {...register('first_name')} />
          <Input label="Last Name" {...register('last_name')} />
        </div>
        <Input label="Email Address" type="email" {...register('email')} disabled />
        <Input label="Phone Number" {...register('phone')} />
        <div className="pt-4">
          <Button type="submit" disabled={!isDirty} loading={updateMutation.isPending}>
            Save Changes
          </Button>
        </div>
      </form>
    </Card>
  );
}

const passwordSchema = z.object({
  current_password: z.string().min(1, 'Current password is required'),
  new_password: z.string().min(8, 'Password must be at least 8 characters'),
  confirm_password: z.string(),
}).refine(data => data.new_password === data.confirm_password, {
  message: 'Passwords do not match',
  path: ['confirm_password'],
});

type PasswordFormData = z.infer<typeof passwordSchema>;

function SecuritySettings() {
  const [isChangingPassword, setIsChangingPassword] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<PasswordFormData>({
    resolver: zodResolver(passwordSchema),
    defaultValues: {
      current_password: '',
      new_password: '',
      confirm_password: '',
    },
  });

  const onSubmit = async (data: PasswordFormData) => {
    setIsChangingPassword(true);
    try {
      await authApi.changePassword(data.current_password, data.new_password);
      toast.success('Password changed successfully');
      reset();
    } catch (error) {
      toast.error(getErrorMessage(error));
    } finally {
      setIsChangingPassword(false);
    }
  };

  return (
    <Card>
      <CardHeader
        title="Security Settings"
        subtitle="Update your password and security preferences"
      />
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <Input
          label="Current Password"
          type="password"
          {...register('current_password')}
          error={errors.current_password?.message}
        />
        <Input
          label="New Password"
          type="password"
          {...register('new_password')}
          error={errors.new_password?.message}
        />
        <Input
          label="Confirm New Password"
          type="password"
          {...register('confirm_password')}
          error={errors.confirm_password?.message}
        />
        <div className="pt-4">
          <Button type="submit" loading={isChangingPassword}>
            Change Password
          </Button>
        </div>
      </form>
    </Card>
  );
}

function NotificationSettings() {
  const { notificationPrefs, setNotificationPref } = useUIStore();

  return (
    <Card>
      <CardHeader title="Notification Preferences" subtitle="Choose how you want to be notified" />
      <div className="space-y-6">
        <div>
          <h4 className="font-medium text-gray-900 mb-3">Notifications</h4>
          <div className="space-y-3">
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Order updates</span>
              <input
                type="checkbox"
                checked={notificationPrefs.orderUpdates}
                onChange={() => setNotificationPref('orderUpdates', !notificationPrefs.orderUpdates)}
                className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              />
            </label>
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Inventory alerts</span>
              <input
                type="checkbox"
                checked={notificationPrefs.inventoryAlerts}
                onChange={() => setNotificationPref('inventoryAlerts', !notificationPrefs.inventoryAlerts)}
                className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              />
            </label>
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-600">System notifications</span>
              <input
                type="checkbox"
                checked={notificationPrefs.systemNotifications}
                onChange={() => setNotificationPref('systemNotifications', !notificationPrefs.systemNotifications)}
                className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              />
            </label>
            <label className="flex items-center justify-between">
              <span className="text-sm text-gray-600">Email notifications</span>
              <input
                type="checkbox"
                checked={notificationPrefs.emailNotifications}
                onChange={() => setNotificationPref('emailNotifications', !notificationPrefs.emailNotifications)}
                className="w-4 h-4 text-primary-600 border-gray-300 rounded focus:ring-primary-500"
              />
            </label>
          </div>
        </div>
      </div>
    </Card>
  );
}

function AppearanceSettings() {
  const { theme, setTheme } = useUIStore();

  return (
    <Card>
      <CardHeader title="Appearance" subtitle="Customize how the admin panel looks" />
      <div className="space-y-4">
        <div>
          <label className="label">Theme</label>
          <div className="grid grid-cols-2 gap-3">
            {(['light', 'dark'] as const).map((t) => (
              <button
                key={t}
                onClick={() => setTheme(t)}
                className={clsx(
                  'p-4 rounded-lg border-2 text-center capitalize transition-colors',
                  theme === t
                    ? 'border-primary-500 bg-primary-50'
                    : 'border-gray-200 hover:border-gray-300'
                )}
              >
                {t}
              </button>
            ))}
          </div>
        </div>
        <p className="text-sm text-gray-500">Note: Dark mode is coming soon!</p>
      </div>
    </Card>
  );
}
