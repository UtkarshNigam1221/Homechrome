import { ArrowUpRightIcon, EnvelopeIcon, PhoneIcon } from '@heroicons/react/24/outline';
import {
  Anchor,
  Box,
  Card,
  Container,
  Group,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from '@mantine/core';
import type { Metadata } from 'next';

import { PageHeader } from '@/components/ui/page-header';
import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import {
  DISPATCH_DAYS,
  GRIEVANCE_OFFICER,
  LEGAL_ADDRESS,
  LEGAL_ENTITY_NAME,
  SUPPORT_EMAIL,
  SUPPORT_PHONE,
  SUPPORT_PHONE_TEL,
  SUPPORT_WHATSAPP,
} from '@/lib/constants';

import { displayFont } from '../fonts';

export const metadata: Metadata = {
  alternates: { canonical: '/contact' },
  title: 'Contact Us | Homechrome',
  description: 'Reach Homechrome support by phone, WhatsApp or email.',
};

// Design tokens: tertiary-fixed for the chat route, primary-fixed for the rest.
const LEAF_TILE = { bg: '#C7ECCE', fg: '#2E4E37' };
const CLAY_TILE = { bg: '#FFDBD0', fg: '#7E2C10' };

export default function ContactPage() {
  const whatsappHref = `https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent('Hi, I need help with my order')}`;

  return (
    <Container size="lg" py="xl">
      <PageHeader
        eyebrow="Support"
        title="Talk to a human"
        description="Questions on a weave, a size, or an order already on its way — reach us however suits you. WhatsApp is fastest."
      />

      <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
        <ContactOption
          href={whatsappHref}
          external
          tile={LEAF_TILE}
          icon={<WhatsAppIcon size={22} />}
          title="WhatsApp"
          detail="Fastest way to reach us"
          action={SUPPORT_PHONE}
        />
        <ContactOption
          href={`tel:${SUPPORT_PHONE_TEL}`}
          tile={CLAY_TILE}
          icon={<PhoneIcon width={22} height={22} aria-hidden="true" />}
          title="Phone"
          detail="Call us directly"
          action={SUPPORT_PHONE}
        />
        <ContactOption
          href={`mailto:${SUPPORT_EMAIL}`}
          tile={CLAY_TILE}
          icon={<EnvelopeIcon width={22} height={22} aria-hidden="true" />}
          title="Email"
          detail="For anything with attachments"
          action={SUPPORT_EMAIL}
        />
      </SimpleGrid>

      <SimpleGrid cols={{ base: 1, md: 2 }} spacing="md" mt="md">
        <Card shadow="sm" radius="lg" padding="lg" withBorder={false} bg="navy.1">
          <Stack gap="sm">
            <SectionTitle>Before you write in</SectionTitle>
            {/* Every line below restates a published policy — keep them in step. */}
            <Fact label="Dispatch">
              Orders leave us within {DISPATCH_DAYS} business days of confirmation.
            </Fact>
            <Fact label="Shipping">Free on every order, to serviceable pincodes across India.</Fact>
            <Fact label="Damage or defect">
              Send an unboxing video and we will replace the piece.
            </Fact>
            <Fact label="Tracking">
              Your order number on the track page shows where a parcel is.
            </Fact>
            <Anchor href="/track" c="brand.5" fw={600} fz="sm" mt={4}>
              <Group gap={6} wrap="nowrap" align="center">
                Track an order
                <ArrowUpRightIcon width={15} height={15} />
              </Group>
            </Anchor>
          </Stack>
        </Card>

        {/* Grievance Officer — required to be displayed under the Consumer
            Protection (E-Commerce) Rules, 2020 and the DPDP Act, 2023. Anchored
            so the footer can deep-link to /contact#grievance. */}
        <Card id="grievance" shadow="sm" radius="lg" padding="lg" withBorder={false}>
          <Stack gap={6}>
            <SectionTitle>Grievance Officer</SectionTitle>
            <Text size="sm" fw={600} c="navy.9">
              {GRIEVANCE_OFFICER}, {LEGAL_ENTITY_NAME}
            </Text>
            <Text size="sm" c="navy.6">
              {LEGAL_ADDRESS}
            </Text>
            <Text size="sm" c="navy.7">
              Phone: <Anchor href={`tel:${SUPPORT_PHONE_TEL}`} c="brand.5">{SUPPORT_PHONE}</Anchor>
            </Text>
            <Text size="sm" c="navy.7">
              Email: <Anchor href={`mailto:${SUPPORT_EMAIL}`} c="brand.5">{SUPPORT_EMAIL}</Anchor>
            </Text>
            <Text size="xs" c="navy.6" mt={4} lh={1.6}>
              Grievances are acknowledged within 48 hours and resolved within one month, as required
              under the Consumer Protection (E-Commerce) Rules, 2020.
            </Text>
          </Stack>
        </Card>
      </SimpleGrid>
    </Container>
  );
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <Title
      order={2}
      fz="1.25rem"
      fw={600}
      c="navy.9"
      style={{ fontFamily: displayFont.style.fontFamily }}
    >
      {children}
    </Title>
  );
}

function Fact({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <Box>
      <Text fz="xs" fw={700} c="navy.5" style={{ letterSpacing: '0.08em' }}>
        {label.toUpperCase()}
      </Text>
      <Text fz="sm" c="navy.7" lh={1.55}>
        {children}
      </Text>
    </Box>
  );
}

function ContactOption({
  href,
  external,
  tile,
  icon,
  title,
  detail,
  action,
}: {
  href: string;
  external?: boolean;
  tile: { bg: string; fg: string };
  icon: React.ReactNode;
  title: string;
  detail: string;
  action: string;
}) {
  return (
    <Anchor
      href={href}
      underline="never"
      h="100%"
      {...(external ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
    >
      <Card shadow="sm" radius="lg" padding="lg" withBorder={false} h="100%">
        <Stack gap="sm" h="100%">
          <Group justify="space-between" align="flex-start" wrap="nowrap">
            <Box
              w={44}
              h={44}
              c={tile.fg}
              style={{
                background: tile.bg,
                borderRadius: 'var(--mantine-radius-md)',
                display: 'grid',
                placeItems: 'center',
              }}
            >
              {icon}
            </Box>
            <ArrowUpRightIcon width={18} height={18} color="var(--mantine-color-navy-5)" />
          </Group>

          <Stack gap={2}>
            <Text fz="1.125rem" fw={600} c="navy.9" style={{ fontFamily: displayFont.style.fontFamily }}>
              {title}
            </Text>
            <Text fz="xs" c="navy.6">
              {detail}
            </Text>
            <Text fz="sm" fw={600} c="brand.5" mt={4} style={{ wordBreak: 'break-word' }}>
              {action}
            </Text>
          </Stack>
        </Stack>
      </Card>
    </Anchor>
  );
}
