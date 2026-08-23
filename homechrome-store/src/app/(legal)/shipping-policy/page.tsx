import { List, ListItem, Text } from '@mantine/core';
import type { Metadata } from 'next';

import { LegalLink } from '@/components/legal/LegalLink';
import { LegalPageLayout, LegalSection } from '@/components/legal/LegalPageLayout';
import { DELIVERY_DAYS, DISPATCH_DAYS } from '@/lib/constants';

export const metadata: Metadata = {
  alternates: { canonical: '/shipping-policy' },
  title: 'Shipping & Delivery Policy | Homechrome',
  description: 'Free shipping across India — dispatch and delivery timelines, tracking, and failed-delivery handling.',
};

export default function ShippingPolicyPage() {
  return (
    <LegalPageLayout title="Shipping & Delivery Policy">
      <LegalSection heading="Shipping charges and coverage">
        <List spacing="xs">
          <ListItem>Shipping is free on all orders.</ListItem>
          <ListItem>We deliver to serviceable pincodes across India only. We do not ship internationally.</ListItem>
        </List>
      </LegalSection>

      <LegalSection heading="Timelines">
        <List spacing="xs">
          <ListItem>Orders are dispatched within {DISPATCH_DAYS} business days of confirmation.</ListItem>
          <ListItem>Delivery takes {DELIVERY_DAYS} business days from dispatch, depending on your location.</ListItem>
          <ListItem>
            Festivals, extreme weather, and remote pincodes can add delays; we will keep you
            informed of any significant change.
          </ListItem>
        </List>
      </LegalSection>

      <LegalSection heading="Tracking">
        <Text>
          Track your order any time at{' '}
          <LegalLink href="/track">homechrome.in/track</LegalLink>{' '}
          using your order ID, or from your account&rsquo;s order history.
        </Text>
      </LegalSection>

      <LegalSection heading="Failed delivery">
        <Text>
          If a delivery attempt fails, the courier will re-attempt or hold the package briefly. If
          your order is returned to us as undeliverable, we will contact you on your registered
          number to arrange re-delivery.
        </Text>
      </LegalSection>

      <LegalSection heading="Damaged packages">
        <Text>
          If the package appears damaged on arrival, record an unboxing video before opening — see
          the{' '}
          <LegalLink href="/refund-policy">Refund, Replacement &amp; Cancellation Policy</LegalLink>{' '}
          for the claim process.
        </Text>
      </LegalSection>
    </LegalPageLayout>
  );
}
