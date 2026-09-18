export type PushSubscriptionStatus = 'ACTIVE' | 'INACTIVE';

export interface PushDeviceInfo {
  browser?: string;
  os?: string;
  is_mobile: boolean;
}

/**
 * A subscriber row. The endpoint's encryption keys are deliberately absent —
 * the backend never serializes them, since endpoint + keys is the full
 * credential needed to push to that device.
 */
export interface PushSubscriber {
  id: string;
  endpoint: string;
  user_agent?: string;
  device?: PushDeviceInfo;
  status: PushSubscriptionStatus;
  created_at: string;
  last_seen_at: string;
}

/** One of the buttons Android shows on an expanded notification. Max two. */
export interface PushAction {
  action: string;
  title: string;
  /** Overrides the notification's click-through for this button. */
  url?: string;
}

export type PushBroadcastStatus = 'SUCCESS' | 'PARTIAL' | 'FAILED';

export interface PushBroadcast {
  id: string;
  title: string;
  body: string;
  url: string;
  tag?: string;
  image?: string;
  icon?: string;
  actions?: PushAction[];
  total_targeted: number;
  success_count: number;
  failure_count: number;
  status: PushBroadcastStatus;
  sent_at: string;
  sent_by: string;
}

export interface BroadcastResult {
  total_targeted: number;
  success_count: number;
  failure_count: number;
  status: PushBroadcastStatus;
  broadcast: PushBroadcast;
}

export interface BroadcastRequest {
  title: string;
  body: string;
  url?: string;
  tag?: string;
  /** Wide banner shown when the notification is expanded. */
  image?: string;
  /** Overrides the brand mark with a product thumbnail. */
  icon?: string;
  actions?: PushAction[];
}
