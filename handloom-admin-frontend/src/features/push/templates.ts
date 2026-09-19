import type { BroadcastRequest } from './types';

/**
 * Starting points for the four broadcasts the storefront actually sends.
 * Every field stays editable — a template fills the form, it does not lock it.
 */
export interface PushTemplate {
  id: string;
  label: string;
  /** What this template is for, so the right one is picked without guessing. */
  purpose: string;
  values: Required<Pick<BroadcastRequest, 'title' | 'body' | 'url'>> &
    Pick<BroadcastRequest, 'image' | 'icon' | 'actions'>;
}

export const PUSH_TEMPLATES: PushTemplate[] = [
  {
    id: 'festive-drop',
    label: 'Festive drop',
    purpose: 'A new collection going live, with a voucher',
    values: {
      title: 'Diwali Heirloom Weaves: 30% Off Starts Today',
      body: 'Hand-carved block prints on Chanderi silk and combed mulmul. Use voucher FIRSTWEAVE at checkout before stocks deplete.',
      url: '/products',
      actions: [
        { action: 'shop', title: 'Shop the loom', url: '/products' },
        { action: 'snooze', title: 'Remind me', url: '/account/reminders' },
      ],
    },
  },
  {
    id: 'cart-reminder',
    label: 'Cart reminder',
    purpose: 'A saved bag left untouched, often with a price drop',
    values: {
      title: 'Price drop: Reversible Mulmul Dohar is now ₹1,199',
      body: 'An item in your saved bag dropped by ₹300. Hand-block printed by Jaipur artisans, with free express delivery.',
      url: '/cart',
      actions: [
        { action: 'checkout', title: 'Checkout', url: '/checkout' },
        { action: 'view_bag', title: 'View bag', url: '/cart' },
      ],
    },
  },
  {
    id: 'order-shipped',
    label: 'Order shipped',
    purpose: 'A dispatch or delivery update for one customer',
    values: {
      title: 'Your order is out for delivery',
      body: 'Your Ditsy Garden Double Bedsheet arrives today. The courier will call before dropping off.',
      url: '/track',
      actions: [{ action: 'track', title: 'Track shipment', url: '/track' }],
    },
  },
  {
    id: 'restock',
    label: 'Back in stock',
    purpose: 'A sold-out piece returning to the shelf',
    values: {
      title: 'Back in stock: Heritage Cushion Collection',
      body: 'Botanical and floral tufted cushions have returned, in limited numbers.',
      url: '/products',
      actions: [{ action: 'discover', title: 'Discover', url: '/products' }],
    },
  },
];
