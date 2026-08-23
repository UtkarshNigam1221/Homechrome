import { EnvelopeIcon, PhoneIcon } from '@heroicons/react/24/outline';
import { Anchor, Card, Container, Group, Stack, Text, ThemeIcon, Title } from '@mantine/core';
import type { Metadata } from 'next';

import { PageHeader } from '@/components/ui/page-header';
import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import {
  GRIEVANCE_OFFICER,
  LEGAL_ADDRESS,
  LEGAL_ENTITY_NAME,
  SUPPORT_EMAIL,
  SUPPORT_PHONE,
  SUPPORT_PHONE_TEL,
  SUPPORT_WHATSAPP,
} from '@/lib/constants';

export const metadata: Metadata = {
  alternates: { canonical: '/contact' },
  title: 'Contact Us | Homechrome',
  description: 'Reach Homechrome support by phone, WhatsApp or email.',
};

export default function ContactPage() {
  const whatsappHref = `https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent('Hi, I need help with my order')}`;

  return (
    <Container size="sm" py="xl">
      <PageHeader
        title="Contact Us"
        description="We're here to help. Call us, or reach us on WhatsApp or email, and we'll get back to you as soon as we can."
      />
      <Stack gap="md">
        <ContactOption
          href={whatsappHref}
          external
          color="green"
          icon={<WhatsAppIcon />}
          title="WhatsApp"
          detail="Chat with us — fastest way to reach support"
        />
        <ContactOption
          href={`tel:${SUPPORT_PHONE_TEL}`}
          color="navy"
          icon={<PhoneIcon width={24} height={24} aria-hidden="true" />}
          title="Phone"
          detail={SUPPORT_PHONE}
        />
        <ContactOption
          href={`mailto:${SUPPORT_EMAIL}`}
          color="brand"
          icon={<EnvelopeIcon width={24} height={24} aria-hidden="true" />}
          title="Email"
          detail={SUPPORT_EMAIL}
        />
      </Stack>

      {/* Grievance Officer — required to be displayed under the Consumer
          Protection (E-Commerce) Rules, 2020 and the DPDP Act, 2023. Anchored
          so the footer can deep-link to /contact#grievance. */}
      <Card id="grievance" shadow="sm" radius="lg" padding="lg" withBorder mt="xl">
        <Stack gap={4}>
          <Title order={2} size="h4" c="navy.7">
            Grievance Officer
          </Title>
          <Text size="sm">
            {GRIEVANCE_OFFICER}, {LEGAL_ENTITY_NAME}
          </Text>
          <Text size="sm" c="dimmed">
            {LEGAL_ADDRESS}
          </Text>
          <Text size="sm">
            Phone: <Anchor href={`tel:${SUPPORT_PHONE_TEL}`}>{SUPPORT_PHONE}</Anchor>
          </Text>
          <Text size="sm">
            Email: <Anchor href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</Anchor>
          </Text>
          <Text size="sm" c="dimmed">
            Grievances are acknowledged within 48 hours and resolved within one month, as required
            under the Consumer Protection (E-Commerce) Rules, 2020.
          </Text>
        </Stack>
      </Card>
    </Container>
  );
}

function ContactOption({
  href,
  external,
  color,
  icon,
  title,
  detail,
}: {
  href: string;
  external?: boolean;
  color: string;
  icon: React.ReactNode;
  title: string;
  detail: string;
}) {
  return (
    <Anchor
      href={href}
      underline="never"
      {...(external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
    >
      <Card shadow="sm" radius="lg" padding="lg" withBorder>
        <Group gap="md" wrap="nowrap">
          <ThemeIcon size={48} radius="md" variant="light" color={color}>
            {icon}
          </ThemeIcon>
          <Stack gap={2}>
            <Text fw={600} c="navy.7">
              {title}
            </Text>
            <Text size="sm" c="dimmed">
              {detail}
            </Text>
          </Stack>
        </Group>
      </Card>
    </Anchor>
  );
}
