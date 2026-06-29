import { EnvelopeIcon } from '@heroicons/react/24/outline';
import { Anchor, Card, Container, Group, Stack, Text, ThemeIcon } from '@mantine/core';

import { PageHeader } from '@/components/ui/page-header';
import { SUPPORT_EMAIL, SUPPORT_WHATSAPP } from '@/lib/constants';

export const metadata = {
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
          icon={<EnvelopeIcon width={24} height={24} />}
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

function WhatsAppIcon() {
  return (
    <svg width={24} height={24} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M19.05 4.91A9.816 9.816 0 0 0 12.04 2c-5.46 0-9.91 4.45-9.91 9.91 0 1.75.46 3.45 1.32 4.95L2.05 22l5.25-1.38c1.45.79 3.08 1.21 4.74 1.21h.01c5.46 0 9.91-4.45 9.91-9.91 0-2.65-1.03-5.14-2.91-7.01zm-7.01 15.24c-1.48 0-2.93-.4-4.2-1.15l-.3-.18-3.12.82.83-3.04-.2-.31a8.264 8.264 0 0 1-1.26-4.38c0-4.54 3.7-8.24 8.25-8.24 2.2 0 4.27.86 5.82 2.42a8.183 8.183 0 0 1 2.41 5.83c0 4.54-3.7 8.23-8.24 8.23zm4.52-6.16c-.25-.12-1.47-.72-1.69-.81-.23-.08-.39-.12-.56.12-.17.25-.64.81-.78.97-.14.17-.29.19-.54.06-.25-.12-1.05-.39-1.99-1.23-.74-.66-1.23-1.47-1.38-1.72-.14-.25-.02-.38.11-.51.11-.11.25-.29.37-.43.13-.14.17-.25.25-.41.08-.17.04-.31-.02-.43-.06-.12-.56-1.34-.76-1.84-.2-.48-.41-.42-.56-.43-.14-.01-.31-.01-.48-.01-.17 0-.43.06-.66.31-.22.25-.86.85-.86 2.07 0 1.22.89 2.4 1.01 2.56.12.17 1.75 2.67 4.23 3.74.59.26 1.05.41 1.41.52.59.19 1.13.16 1.56.1.48-.07 1.47-.6 1.68-1.18.21-.58.21-1.07.14-1.18-.06-.11-.22-.17-.47-.29z" />
    </svg>
  );
}
