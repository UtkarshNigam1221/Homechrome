import { EnvelopeIcon } from '@heroicons/react/24/outline';
import { Anchor, Card, Container, Group, Stack, Text, ThemeIcon } from '@mantine/core';
import type { Metadata } from 'next';

import { PageHeader } from '@/components/ui/page-header';
import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import { SUPPORT_EMAIL, SUPPORT_WHATSAPP } from '@/lib/constants';

export const metadata: Metadata = {
  title: 'Contact Us | Homechrome',
  description: 'Reach Homechrome support on WhatsApp or by email.',
};

export default function ContactPage() {
  const whatsappHref = `https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent('Hi, I need help with my order')}`;

  return (
    <Container size="sm" py="xl">
      <PageHeader
        title="Contact Us"
        description="We're here to help. Reach us on WhatsApp or email and we'll get back to you as soon as we can."
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
          href={`mailto:${SUPPORT_EMAIL}`}
          color="brand"
          icon={<EnvelopeIcon width={24} height={24} aria-hidden="true" />}
          title="Email"
          detail={SUPPORT_EMAIL}
        />
      </Stack>
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
