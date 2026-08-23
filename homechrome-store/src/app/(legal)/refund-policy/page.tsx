import { Anchor, List, ListItem, Text } from '@mantine/core';
import type { Metadata } from 'next';

import { LegalLink } from '@/components/legal/LegalLink';
import { LegalPageLayout, LegalSection } from '@/components/legal/LegalPageLayout';
import { DAMAGE_CLAIM_WINDOW_HOURS, REFUND_DAYS, SUPPORT_EMAIL } from '@/lib/constants';

export const metadata: Metadata = {
  alternates: { canonical: '/refund-policy' },
  title: 'Refund, Replacement & Cancellation Policy | Homechrome',
  description:
    'Replacement for damaged or defective items, unboxing-video requirement, claim window, and order cancellation terms.',
};

export default function RefundPolicyPage() {
  return (
    <LegalPageLayout title="Refund, Replacement & Cancellation Policy">
      <LegalSection heading="Summary">
        <Text>
          We offer <Text span fw={600}>replacements only</Text>, for items that arrive damaged or
          with a manufacturing defect. We do not offer returns or refunds for change of mind. A
          continuous unboxing video is mandatory for every claim.
        </Text>
      </LegalSection>

      <LegalSection heading="Replacement eligibility">
        <List spacing="xs">
          <ListItem>The item arrived physically damaged in transit; or</ListItem>
          <ListItem>the item has a genuine manufacturing defect.</ListItem>
        </List>
        <Text>
          Minor variations in colour, weave, texture, or dimension are inherent to handloom
          products and are not defects (see our{' '}
          <LegalLink href="/terms">Terms &amp; Conditions</LegalLink>
          ).
        </Text>
      </LegalSection>

      <LegalSection heading="Unboxing video (mandatory)">
        <Text>
          Record a single, uncut video that starts with the sealed, unopened package (shipping
          label visible) and continues through opening to clearly show the damage or defect.
          Claims without a qualifying unboxing video cannot be accepted — this protects both you
          and us.
        </Text>
      </LegalSection>

      <LegalSection heading="How to claim">
        <List spacing="xs" type="ordered">
          <ListItem>
            Contact us within {DAMAGE_CLAIM_WINDOW_HOURS} hours of delivery — on WhatsApp or at{' '}
            <Anchor href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</Anchor> — with your order ID,
            photos, and the unboxing video.
          </ListItem>
          <ListItem>We review and respond within 2 business days.</ListItem>
          <ListItem>
            If approved, a replacement ships free of charge. If the item is out of stock, we refund
            the full amount to your original payment method via PhonePe within {REFUND_DAYS}{' '}
            business days.
          </ListItem>
        </List>
        <Text>Claims made after {DAMAGE_CLAIM_WINDOW_HOURS} hours of delivery cannot be accepted.</Text>
      </LegalSection>

      <LegalSection heading="Cancellation">
        <List spacing="xs">
          <ListItem>
            <Text span fw={600}>
              Before dispatch:
            </Text>{' '}
            you may cancel for a full refund to your original payment method (processed within{' '}
            {REFUND_DAYS} business days). Contact support with your order ID.
          </ListItem>
          <ListItem>
            <Text span fw={600}>
              After dispatch:
            </Text>{' '}
            orders cannot be cancelled.
          </ListItem>
        </List>
      </LegalSection>

      <LegalSection heading="Questions">
        <Text>
          Contact us via the{' '}
          <LegalLink href="/contact">contact page</LegalLink>{' '}
          — WhatsApp is fastest.
        </Text>
      </LegalSection>
    </LegalPageLayout>
  );
}
