import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { clsx } from 'clsx';
import { format } from 'date-fns';
import { AlertCircle, Bell, CheckCheck, Package, ShoppingCart, Tag } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { notificationsApi } from '@/features/notifications/api';
import { getErrorMessage } from '@/shared/api/client';
import { PageLoading } from '@/shared/components/loading';
import { Badge, Button, Card } from '@/shared/components/ui';

import type { NotificationType } from '../types';

const getNotificationIcon = (type: NotificationType) => {
  switch (type) {
    case 'ORDER':
      return <ShoppingCart className="w-5 h-5" />;
    case 'PRODUCT':
      return <Package className="w-5 h-5" />;
    case 'ALERT':
      return <AlertCircle className="w-5 h-5" />;
    case 'PROMOTION':
      return <Tag className="w-5 h-5" />;
    default:
      return <Bell className="w-5 h-5" />;
  }
};

const getTypeColor = (type: NotificationType) => {
  switch (type) {
    case 'ORDER':
      return 'bg-blue-100 text-blue-600';
    case 'PRODUCT':
      return 'bg-green-100 text-green-600';
    case 'ALERT':
      return 'bg-red-100 text-red-600';
    case 'PROMOTION':
      return 'bg-purple-100 text-purple-600';
    default:
      return 'bg-gray-100 text-gray-600';
  }
};

export function NotificationsPage() {
  const queryClient = useQueryClient();
  const [filter, setFilter] = useState<'all' | 'unread'>('all');

  const { data, isLoading } = useQuery({
    queryKey: ['my-notifications'],
    queryFn: () => notificationsApi.getMy({ limit: 50 }),
  });

  const markAsReadMutation = useMutation({
    mutationFn: notificationsApi.markAsRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-notifications'] });
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const markAllAsReadMutation = useMutation({
    mutationFn: notificationsApi.markAllAsRead,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-notifications'] });
      toast.success('All notifications marked as read');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  if (isLoading) {
    return <PageLoading />;
  }

  const notifications = data?.items || [];
  const unreadCount = notifications.filter((n) => n.status === 'UNREAD').length;
  const filteredNotifications =
    filter === 'unread' ? notifications.filter((n) => n.status === 'UNREAD') : notifications;

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="page-title">Notifications</h1>
          <p className="page-subtitle">
            {unreadCount > 0 ? `You have ${unreadCount} unread notifications` : 'All caught up!'}
          </p>
        </div>
        {unreadCount > 0 && (
          <Button
            variant="secondary"
            leftIcon={<CheckCheck className="w-4 h-4" />}
            onClick={() => markAllAsReadMutation.mutate()}
            loading={markAllAsReadMutation.isPending}
          >
            Mark all as read
          </Button>
        )}
      </div>

      {/* Filter Tabs */}
      <div className="flex gap-2">
        <button
          onClick={() => setFilter('all')}
          className={clsx(
            'px-4 py-2 text-sm font-medium rounded-lg transition-colors',
            filter === 'all' ? 'bg-primary-100 text-primary-700' : 'text-gray-600 hover:bg-gray-100'
          )}
        >
          All ({notifications.length})
        </button>
        <button
          onClick={() => setFilter('unread')}
          className={clsx(
            'px-4 py-2 text-sm font-medium rounded-lg transition-colors',
            filter === 'unread'
              ? 'bg-primary-100 text-primary-700'
              : 'text-gray-600 hover:bg-gray-100'
          )}
        >
          Unread ({unreadCount})
        </button>
      </div>

      {/* Notifications List */}
      <Card padding="none">
        {filteredNotifications.length === 0 ? (
          <div className="p-12 text-center">
            <Bell className="w-12 h-12 mx-auto text-gray-300 mb-4" />
            <p className="text-gray-500">
              {filter === 'unread' ? 'No unread notifications' : 'No notifications yet'}
            </p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {filteredNotifications.map((notification) => (
              <div
                key={notification.id}
                className={clsx(
                  'p-4 hover:bg-gray-50 transition-colors cursor-pointer',
                  notification.status === 'UNREAD' && 'bg-primary-50/30'
                )}
                onClick={() => {
                  if (notification.status === 'UNREAD') {
                    markAsReadMutation.mutate(notification.id);
                  }
                }}
              >
                <div className="flex gap-4">
                  <div className={clsx('p-2 rounded-lg', getTypeColor(notification.type))}>
                    {getNotificationIcon(notification.type)}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <p
                          className={clsx(
                            'text-sm',
                            notification.status === 'UNREAD'
                              ? 'font-semibold text-gray-900'
                              : 'font-medium text-gray-700'
                          )}
                        >
                          {notification.title}
                        </p>
                        <p className="text-sm text-gray-600 mt-1">{notification.message}</p>
                      </div>
                      {notification.status === 'UNREAD' && (
                        <span className="w-2 h-2 bg-primary-500 rounded-full flex-shrink-0 mt-2" />
                      )}
                    </div>
                    <div className="flex items-center gap-2 mt-2">
                      <Badge variant="gray" size="sm">
                        {notification.type}
                      </Badge>
                      <span className="text-xs text-gray-400">
                        {format(new Date(notification.created_at), 'MMM d, yyyy HH:mm')}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
