import { Anchor, List, ListItem, Text } from '@mantine/core';
import type { Metadata } from 'next';

import { LegalPageLayout, LegalSection } from '@/components/legal/LegalPageLayout';
import {
  GRIEVANCE_OFFICER,
  LEGAL_ADDRESS,
  LEGAL_ENTITY_NAME,
  LEGAL_PROPRIETOR,
  SUPPORT_EMAIL,
} from '@/lib/constants';

export const metadata: Metadata = {
  alternates: { canonical: '/privacy-policy' },
  title: 'Privacy Policy | Homechrome',
  description:
    'How Homechrome collects, uses, and protects your personal data, and your rights under the DPDP Act, 2023.',
};

export default function PrivacyPolicyPage() {
  return (
    <LegalPageLayout title="Privacy Policy">
      <LegalSection heading="Who we are">
        <Text>
          www.homechrome.in (&ldquo;Homechrome&rdquo;, &ldquo;we&rdquo;, &ldquo;us&rdquo;) is
          operated by {LEGAL_ENTITY_NAME}, a sole proprietorship firm (proprietor:{' '}
          {LEGAL_PROPRIETOR}), having its registered address at {LEGAL_ADDRESS}. This policy
          explains what personal data we collect, why we collect it, and the rights you have over
          it, in accordance with the Information Technology Act, 2000 and the Digital Personal Data
          Protection Act, 2023 (&ldquo;DPDP Act&rdquo;).
        </Text>
      </LegalSection>

      <LegalSection heading="What we collect">
        <List spacing="xs">
          <ListItem>
            <Text span fw={600}>
              Account data:
            </Text>{' '}
            your mobile number (used for OTP login), your name, and delivery addresses you save.
          </ListItem>
          <ListItem>
            <Text span fw={600}>
              Order data:
            </Text>{' '}
            products ordered, order value, delivery and payment status, and order history.
          </ListItem>
          <ListItem>
            <Text span fw={600}>
              Analytics data:
            </Text>{' '}
            device type, approximate location, marketing attribution tags (UTM parameters), an
            anonymous visitor identifier stored in your browser&rsquo;s local storage, and
            aggregate browsing events (such as pages viewed and items added to cart).
          </ListItem>
          <ListItem>
            <Text span fw={600}>
              Cookies:
            </Text>{' '}
            we use functional cookies only — secure, HttpOnly session cookies that keep you signed
            in. We do not use third-party advertising or cross-site tracking cookies.
          </ListItem>
        </List>
      </LegalSection>

      <LegalSection heading="How we use your data">
        <List spacing="xs">
          <ListItem>To create and secure your account via one-time passwords (OTP).</ListItem>
          <ListItem>To process, deliver, and provide support for your orders.</ListItem>
          <ListItem>To communicate order updates on your registered contact details.</ListItem>
          <ListItem>
            To understand, in aggregate, how visitors find and use our store so we can improve it.
          </ListItem>
        </List>
        <Text>
          We do not sell or rent your personal data to anyone, and we do not use it for third-party
          advertising.
        </Text>
      </LegalSection>

      <LegalSection heading="Who we share it with">
        <List spacing="xs">
          <ListItem>
            <Text span fw={600}>
              MSG91
            </Text>{' '}
            — to deliver OTP SMS messages to your mobile number.
          </ListItem>
          <ListItem>
            <Text span fw={600}>
              PhonePe
            </Text>{' '}
            — to process payments. Your card, UPI, and banking details are handled entirely by
            PhonePe; we never receive or store them.
          </ListItem>
          <ListItem>
            <Text span fw={600}>
              Amazon Web Services (Mumbai region, India)
            </Text>{' '}
            — our hosting and data-storage provider. Your data is stored in India.
          </ListItem>
          <ListItem>Delivery partners — your name, address, and phone number, solely to deliver your order.</ListItem>
        </List>
      </LegalSection>

      <LegalSection heading="How long we keep it">
        <Text>
          Account data is retained while your account remains active. Order and payment records are
          retained as required for accounting, tax, and other legal obligations, after which they
          are deleted or anonymised.
        </Text>
      </LegalSection>

      <LegalSection heading="Your rights">
        <Text>
          Under the DPDP Act, 2023 you may request access to the personal data we hold about you,
          correction of inaccurate data, erasure of data we are not legally required to retain, and
          you may raise a grievance about how your data is handled. To exercise any of these
          rights, write to{' '}
          <Anchor href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</Anchor> from your registered
          contact details.
        </Text>
      </LegalSection>

      <LegalSection heading="Grievance Officer">
        <Text>
          {GRIEVANCE_OFFICER}, {LEGAL_ENTITY_NAME}, {LEGAL_ADDRESS}. Email:{' '}
          <Anchor href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</Anchor>. Grievances are
          acknowledged within 48 hours and resolved within one month.
        </Text>
      </LegalSection>

      <LegalSection heading="Changes to this policy">
        <Text>
          We may update this policy from time to time. The &ldquo;Last updated&rdquo; date above
          reflects the latest revision; continued use of the site after an update constitutes
          acceptance of the revised policy.
        </Text>
      </LegalSection>
    </LegalPageLayout>
  );
}
