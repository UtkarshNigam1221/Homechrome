export type NotificationType = 'ORDER' | 'PRODUCT' | 'SYSTEM' | 'PROMOTION' | 'ALERT';
export type NotificationStatus = 'UNREAD' | 'READ' | 'ARCHIVED';

export interface Notification {
  id: string;
  user_id: string;
  type: NotificationType;
  title: string;
  message: string;
  data?: Record<string, unknown>;
  priority?: 'low' | 'normal' | 'high';
  status: NotificationStatus;
  read_at?: string;
  created_at: string;
}
