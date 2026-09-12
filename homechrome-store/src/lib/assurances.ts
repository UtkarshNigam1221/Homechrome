import {
  ClockIcon,
  LockClosedIcon,
  ShieldCheckIcon,
  TruckIcon,
} from '@heroicons/react/24/outline';

import { DAMAGE_CLAIM_WINDOW_HOURS, DELIVERY_DAYS, DISPATCH_DAYS } from './constants';

/**
 * What the store commits to, stated once. Every one of these restates a
 * published policy — do not add a promise the policy pages do not make, and
 * when a policy changes, change it here rather than in each screen.
 */
export const ASSURANCES = [
  {
    key: 'shipping',
    icon: TruckIcon,
    title: 'Free shipping, always',
    body: 'Free on every order, to serviceable pincodes across India.',
    short: 'On every order, across India',
  },
  {
    key: 'dispatch',
    icon: ClockIcon,
    title: `Dispatched in ${DISPATCH_DAYS} days`,
    body: `Delivery takes a further ${DELIVERY_DAYS} business days.`,
    short: `Then ${DELIVERY_DAYS} days in transit`,
  },
  {
    key: 'replacement',
    icon: ShieldCheckIcon,
    title: 'Damage replaced',
    body: `Send an unboxing video within ${DAMAGE_CLAIM_WINDOW_HOURS} hours and we replace the piece.`,
    short: `Unboxing video within ${DAMAGE_CLAIM_WINDOW_HOURS} hours`,
  },
  {
    key: 'payments',
    icon: LockClosedIcon,
    title: 'Secure payments',
    body: 'Every checkout runs through an encrypted, PCI-compliant gateway.',
    short: 'Encrypted PhonePe checkout',
  },
] as const;

export type Assurance = (typeof ASSURANCES)[number];
