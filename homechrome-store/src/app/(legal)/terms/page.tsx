import { Anchor, List, ListItem, Text } from '@mantine/core';
import type { Metadata } from 'next';

import { LegalLink } from '@/components/legal/LegalLink';
import { LegalPageLayout, LegalSection } from '@/components/legal/LegalPageLayout';
import { LEGAL_ADDRESS, LEGAL_ENTITY_NAME, LEGAL_PROPRIETOR, SUPPORT_EMAIL } from '@/lib/constants';

export const metadata: Metadata = {
  title: 'Terms & Conditions | Homechrome',
  description: 'Terms and conditions governing the use of www.homechrome.in.',
};

export default function TermsPage() {
  return (
    <LegalPageLayout title="Terms & Conditions">
      <LegalSection heading="About us">
        <Text>
          www.homechrome.in (the &ldquo;Site&rdquo;) is owned and operated by {LEGAL_ENTITY_NAME},
          a sole proprietorship firm (proprietor: {LEGAL_PROPRIETOR}), registered at{' '}
          {LEGAL_ADDRESS} ([GSTIN to be updated]). By using the Site you agree to these Terms; if
          you do not agree, please do not use the Site.
        </Text>
      </LegalSection>

      <LegalSection heading="Your account">
        <Text>
          Sign-in is via a one-time password (OTP) sent to your mobile number. You are responsible
          for the security of your phone number and device; orders and actions taken after a
          successful OTP sign-in are treated as yours. You must be capable of entering into a
          contract under Indian law to place an order.
        </Text>
      </LegalSection>

      <LegalSection heading="Products and handloom disclaimer">
        <Text>
          Our products are handloom and handcrafted. Minor variations in colour, weave, texture,
          and dimension are inherent characteristics of handmade goods — they are a mark of
          authenticity, not a defect. Product photographs are as accurate as reasonably possible,
          but on-screen colours may differ from the physical product depending on your display.
        </Text>
      </LegalSection>

      <LegalSection heading="Pricing and orders">
        <List spacing="xs">
          <ListItem>All prices are in Indian Rupees (INR) and inclusive of applicable taxes.</ListItem>
          <ListItem>
            Placing an order constitutes an offer to purchase. Our acceptance occurs when the order
            is dispatched; we may cancel and fully refund any order before dispatch (for example,
            in case of stock or listing errors).
          </ListItem>
          <ListItem>
            Obvious pricing or description errors may be corrected, and affected orders cancelled
            and refunded, at any time before dispatch.
          </ListItem>
          <ListItem>Payments are processed by PhonePe. We do not store your payment instrument details.</ListItem>
        </List>
      </LegalSection>

      <LegalSection heading="Shipping, cancellation, and replacement">
        <Text>
          Shipping terms are set out in the{' '}
          <LegalLink href="/shipping-policy">Shipping &amp; Delivery Policy</LegalLink>{' '}
          and replacement/cancellation terms in the{' '}
          <LegalLink href="/refund-policy">Refund, Replacement &amp; Cancellation Policy</LegalLink>
          . Both form part of these Terms.
        </Text>
      </LegalSection>

      <LegalSection heading="Intellectual property">
        <Text>
          &ldquo;HOME CHROME&rdquo;&trade; (trademark application pending) and all Site content —
          logos, images, text, and design — are the property of {LEGAL_ENTITY_NAME}. You may not
          reproduce or use them without prior written consent.
        </Text>
      </LegalSection>

      <LegalSection heading="Limitation of liability">
        <Text>
          To the maximum extent permitted by law, our total liability for any claim arising out of
          an order is limited to the amount paid for that order. We are not liable for indirect or
          consequential losses. Nothing in these Terms limits rights that cannot be waived under
          the Consumer Protection Act, 2019.
        </Text>
      </LegalSection>

      <LegalSection heading="Force majeure">
        <Text>
          We are not responsible for delay or failure to perform caused by events beyond our
          reasonable control, including natural calamities, strikes, transport disruptions, or
          government restrictions.
        </Text>
      </LegalSection>

      <LegalSection heading="Governing law and jurisdiction">
        <Text>
          These Terms are governed by the laws of India. Subject to any non-excludable consumer
          rights, courts at Hapur, Uttar Pradesh shall have exclusive jurisdiction.
        </Text>
      </LegalSection>

      <LegalSection heading="Contact">
        <Text>
          Questions about these Terms: <Anchor href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</Anchor>.
        </Text>
      </LegalSection>
    </LegalPageLayout>
  );
}
