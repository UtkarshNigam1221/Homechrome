import {
  BuildingStorefrontIcon,
  ChatBubbleLeftRightIcon,
  EnvelopeIcon,
  ScaleIcon,
  ShieldCheckIcon,
  TruckIcon,
} from '@heroicons/react/24/outline';
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

import { AssuranceRow } from '@/components/ui/assurance-row';
import { Breadcrumb } from '@/components/ui/breadcrumb';
import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import {
  DAMAGE_CLAIM_WINDOW_HOURS,
  DELIVERY_DAYS,
  DISPATCH_DAYS,
  GRIEVANCE_OFFICER,
  INSTAGRAM_URL,
  LEGAL_ADDRESS,
  LEGAL_ENTITY_NAME,
  SUPPORT_EMAIL,
  SUPPORT_PHONE,
  SUPPORT_PHONE_TEL,
} from '@/lib/constants';
import { whatsappHref } from '@/lib/whatsapp';

import { ContactForm } from './ContactForm';
import { ContactFaq } from './ContactFaq';
import { DeskCard, ReachTile } from './ContactCards';

export const metadata: Metadata = {
  alternates: { canonical: '/contact' },
  title: 'Contact Us | Homechrome',
  description: 'Reach Homechrome support by WhatsApp, phone or email.',
};

const LEAF = { bg: 'var(--mantine-color-leaf-1)', fg: 'var(--mantine-color-leaf-6)' };
const CLAY = { bg: 'var(--mantine-color-brand-1)', fg: 'var(--mantine-color-brand-6)' };
const MIST = { bg: 'var(--mantine-color-mist-1)', fg: 'var(--mantine-color-mist-6)' };
const SAND = { bg: 'var(--mantine-color-navy-3)', fg: 'var(--mantine-color-navy-6)' };

const FAQS = [
  {
    q: 'How do I track my order?',
    a: `Enter your order number on the track page and you will see where the parcel is. Orders leave us within ${DISPATCH_DAYS} business days of confirmation, and delivery takes a further ${DELIVERY_DAYS} business days depending on your pincode.`,
  },
  {
    q: 'What is your return and exchange policy?',
    a: `We replace pieces that arrive damaged or with a manufacturing defect — record an unboxing video and send it to us within ${DAMAGE_CLAIM_WINDOW_HOURS} hours of delivery. We do not offer returns or refunds for a change of mind, so do check dimensions before ordering.`,
  },
  {
    q: 'Is shipping really free?',
    a: 'Yes — shipping is free on all orders, with no minimum. We deliver to serviceable pincodes across India only.',
  },
  {
    q: 'Can I order a custom size?',
    a: 'Some pieces can be made to custom dimensions; their product page says so. Message us with the measurements you need and we will tell you whether that weave can be made to size.',
  },
];

export default function ContactPage() {
  return (
    <>
      <Container size="xl" py="xl">
        <Breadcrumb items={[{ label: 'Home', href: '/' }, { label: 'Contact Us' }]} />

        <Stack gap="md" mb="xl">
          <Group
            gap={8}
            w="fit-content"
            px={12}
            py={6}
            bg="navy.2"
            style={{ borderRadius: 999 }}
          >
            <Box w={7} h={7} bg="brand.5" style={{ borderRadius: 999 }} />
            <Text fz={11} fw={700} c="navy.7" style={{ letterSpacing: '0.12em' }}>
              HANDLOOM SUPPORT DESK
            </Text>
          </Group>

          <Title
            order={1}
            fz={{ base: '2.25rem', sm: '3rem' }}
            fw={600}
            lh={1.1}
            c="navy.9"
          >
            Connect with the{' '}
            <Text span inherit c="brand.5" fs="italic">
              Loom
            </Text>
          </Title>

          <Text c="navy.6" maw={640} lh={1.65}>
            Questions about a weave, a size, or an order already on its way — reach us however
            suits you. WhatsApp is the fastest, and a real person answers.
          </Text>
        </Stack>

        <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="md">
          <DeskCard
            tile={LEAF}
            icon={<WhatsAppIcon size={20} />}
            eyebrow="Fastest route"
            title="WhatsApp Support"
            body="Ask about a fabric, a size, or an order in transit and get a reply in the same thread."
            detail={SUPPORT_PHONE}
            action={{
              label: 'Start WhatsApp chat',
              href: whatsappHref('Hi, I need help with my order'),
              external: true,
              tone: 'leaf',
            }}
          />
          <DeskCard
            tile={CLAY}
            icon={<TruckIcon width={20} height={20} />}
            eyebrow="Dispatch"
            title="Order Tracking"
            body={`Orders leave us in ${DISPATCH_DAYS} business days, then ${DELIVERY_DAYS} days in transit.`}
            detail={SUPPORT_EMAIL}
            action={{ label: 'Track an order', href: '/track' }}
          />
          <DeskCard
            tile={MIST}
            icon={<ScaleIcon width={20} height={20} />}
            eyebrow="Consumer compliance"
            title="Grievance Officer"
            body="Formal escalation under the Consumer Protection (E-Commerce) Rules, 2020. Acknowledged in 48 hours."
            detail={`${GRIEVANCE_OFFICER}, ${LEGAL_ENTITY_NAME}`}
            action={{ label: 'Escalation details', href: '#grievance' }}
          />
          <DeskCard
            tile={SAND}
            icon={<BuildingStorefrontIcon width={20} height={20} />}
            eyebrow="Registered office"
            title={LEGAL_ENTITY_NAME}
            body={LEGAL_ADDRESS}
            detail="Correspondence address"
          />
        </SimpleGrid>

        <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg" mt="lg">
          <Card shadow="sm" radius="lg" padding="xl" withBorder={false}>
            <ContactForm />
          </Card>

          <Stack gap="lg">
            <Card shadow="sm" radius="lg" padding="lg" withBorder={false} bg="navy.1">
              <Stack gap="md">
                <Group gap={8} wrap="nowrap" align="center">
                  <ShieldCheckIcon width={18} height={18} color="var(--mantine-color-brand-6)" />
                  <Stack gap={0}>
                    <Text fz={11} fw={700} c="brand.5" style={{ letterSpacing: '0.12em' }}>
                      WHAT WE COMMIT TO
                    </Text>
                    <Title
                      order={2}
                      fz="1.25rem"
                      fw={600}
                      c="navy.9"
                    >
                      The Homechrome Promise
                    </Title>
                  </Stack>
                </Group>

                <AssuranceRow columns={1} />
              </Stack>
            </Card>

            <SimpleGrid cols={{ base: 1, xs: 2 }} spacing="md">
              <ReachTile
                href={`tel:${SUPPORT_PHONE_TEL}`}
                tile={CLAY}
                icon={<ChatBubbleLeftRightIcon width={18} height={18} />}
                label="SUPPORT LINE"
                value={SUPPORT_PHONE}
              />
              <ReachTile
                href={`mailto:${SUPPORT_EMAIL}`}
                tile={MIST}
                icon={<EnvelopeIcon width={18} height={18} />}
                label="EMAIL"
                value={SUPPORT_EMAIL}
              />
            </SimpleGrid>

            {/* Grievance Officer — required under the Consumer Protection
                (E-Commerce) Rules, 2020 and the DPDP Act, 2023. Anchored so the
                footer can deep-link to /contact#grievance. */}
            <Card id="grievance" shadow="sm" radius="lg" padding="lg" withBorder={false}>
              <Stack gap={6}>
                <Title
                  order={2}
                  fz="1.25rem"
                  fw={600}
                  c="navy.9"
                >
                  Grievance Officer
                </Title>
                <Text fz="sm" fw={600} c="navy.9">
                  {GRIEVANCE_OFFICER}, {LEGAL_ENTITY_NAME}
                </Text>
                <Text fz="sm" c="navy.6">
                  {LEGAL_ADDRESS}
                </Text>
                <Text fz="sm" c="navy.7">
                  Phone:{' '}
                  <Anchor href={`tel:${SUPPORT_PHONE_TEL}`} c="brand.5">
                    {SUPPORT_PHONE}
                  </Anchor>
                </Text>
                <Text fz="sm" c="navy.7">
                  Email:{' '}
                  <Anchor href={`mailto:${SUPPORT_EMAIL}`} c="brand.5">
                    {SUPPORT_EMAIL}
                  </Anchor>
                </Text>
                <Text fz="xs" c="navy.6" mt={4} lh={1.6}>
                  Grievances are acknowledged within 48 hours and resolved within one month, as
                  required under the Consumer Protection (E-Commerce) Rules, 2020.
                </Text>
              </Stack>
            </Card>
          </Stack>
        </SimpleGrid>
      </Container>

      <Box component="section" bg="navy.1" py={{ base: 48, sm: 64 }}>
        <Container size="md">
          <Stack gap="xs" align="center" mb="xl">
            <Text fz={11} fw={700} c="brand.5" style={{ letterSpacing: '0.14em' }}>
              INSTANT CLARITY
            </Text>
            <Title
              order={2}
              fz={{ base: '1.75rem', sm: '2.25rem' }}
              fw={600}
              c="navy.9"
              ta="center"
            >
              Frequently Asked Questions
            </Title>
            <Text fz="sm" c="navy.6" ta="center" maw={520}>
              Dispatch timelines, replacements, shipping and custom sizes — the answers we give
              most often.
            </Text>
          </Stack>

          <ContactFaq faqs={FAQS} />

          <Text fz="sm" c="navy.6" ta="center" mt="xl">
            Still need a hand? Call{' '}
            <Anchor href={`tel:${SUPPORT_PHONE_TEL}`} c="brand.5" fw={600}>
              {SUPPORT_PHONE}
            </Anchor>
            , message us on{' '}
            <Anchor href={whatsappHref('Hi, I need help with my order')} target="_blank" rel="noopener noreferrer" c="brand.5" fw={600}>
              WhatsApp
            </Anchor>
            , or find us on{' '}
            <Anchor href={INSTAGRAM_URL} target="_blank" rel="noopener noreferrer" c="brand.5" fw={600}>
              Instagram
            </Anchor>
            .
          </Text>
        </Container>
      </Box>
    </>
  );
}
