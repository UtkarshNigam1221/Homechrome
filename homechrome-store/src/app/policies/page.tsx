import {
  BanknotesIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
  LockClosedIcon,
  ShieldCheckIcon,
  SwatchIcon,
  TruckIcon,
  VideoCameraIcon,
} from '@heroicons/react/24/outline';
import { Box, Card, Container, Flex, Group, SimpleGrid, Stack, Text, Title } from '@mantine/core';
import type { Metadata } from 'next';

import { Breadcrumb } from '@/components/ui/breadcrumb';
import {
  DAMAGE_CLAIM_WINDOW_HOURS,
  DELIVERY_DAYS,
  DISPATCH_DAYS,
  REFUND_DAYS,
} from '@/lib/constants';

import { PoliciesNav } from './PoliciesNav';
import { displayFont } from '../fonts';

export const metadata: Metadata = {
  alternates: { canonical: '/policies' },
  title: 'Policies | Homechrome',
  description:
    'Replacement, shipping, terms, privacy and grievance policies for Homechrome handloom textiles.',
};

// Each step restates the refund policy. Keep them in step with that page.
const STEPS = [
  {
    n: 1,
    icon: VideoCameraIcon,
    title: 'Send the unboxing video',
    body: `Within ${DAMAGE_CLAIM_WINDOW_HOURS} hours of delivery, message us your order ID, photos and a single uncut video that starts at the sealed package and continues through opening.`,
    foot: `Claims after ${DAMAGE_CLAIM_WINDOW_HOURS} hours cannot be accepted`,
  },
  {
    n: 2,
    icon: CheckCircleIcon,
    title: 'We review it',
    body: 'We look at the video and photos and come back to you with a decision. No courier pickup is arranged before that.',
    foot: 'Reviewed within 2 business days',
  },
  {
    n: 3,
    icon: BanknotesIcon,
    title: 'Replacement or refund',
    body: `If approved, a replacement ships free. If the piece is out of stock, the full amount is refunded to your original payment method.`,
    foot: `Refunds land in ${REFUND_DAYS} business days`,
  },
];

// Straight from the refund policy: these are handloom characteristics, not faults.
const VARIATIONS = [
  {
    title: 'Colour',
    body: 'Natural dye lots shift between batches, so two pieces of the same design can read slightly differently.',
  },
  {
    title: 'Weave',
    body: 'Small irregularities in tightness are what separates a hand-set loom from a machine.',
  },
  {
    title: 'Texture',
    body: 'Hand-finished cloth softens and settles over its first few washes.',
  },
  {
    title: 'Dimension',
    body: 'Cut and finish vary a little by hand, so measurements carry a small tolerance.',
  },
];

const NOT_COVERED = [
  {
    title: 'A change of mind',
    body: 'We do not accept returns or refunds because a piece was no longer wanted. Do check dimensions before ordering.',
  },
  {
    title: 'Claims after the window',
    body: `Anything raised more than ${DAMAGE_CLAIM_WINDOW_HOURS} hours after delivery.`,
  },
  {
    title: 'Claims without the video',
    body: 'A continuous unboxing video is required for every claim — it protects you as much as us.',
  },
  {
    title: 'Natural variation',
    body: 'The colour, weave, texture and dimension differences described above are not defects.',
  },
];

const ASSURANCES = [
  { icon: TruckIcon, title: 'Free shipping', body: 'On every order, across India' },
  { icon: ShieldCheckIcon, title: 'Damage replaced', body: 'Free replacement when approved' },
  { icon: LockClosedIcon, title: 'Secure payments', body: 'Encrypted PhonePe checkout' },
  { icon: SwatchIcon, title: 'Handloom textiles', body: 'Woven and printed in India' },
];

export default function PoliciesPage() {
  return (
    <>
      <Box bg="navy.1" py={{ base: 32, sm: 48 }}>
        <Container size="xl">
          <Breadcrumb items={[{ label: 'Home', href: '/' }, { label: 'Policies' }]} />

          <Stack gap="md" maw={720}>
            <Group gap={8} w="fit-content" px={12} py={6} bg="brand.1" style={{ borderRadius: 999 }}>
              <Text fz={11} fw={700} c="brand.6" style={{ letterSpacing: '0.12em' }}>
                WHAT WE STAND BEHIND
              </Text>
            </Group>

            <Title
              order={1}
              fz={{ base: '2rem', sm: '2.75rem' }}
              fw={600}
              lh={1.12}
              c="navy.9"
              style={{ fontFamily: displayFont.style.fontFamily }}
            >
              Our Handloom Promise &amp;{' '}
              <Text span inherit c="brand.5" fs="italic">
                Store Policies
              </Text>
            </Title>

            <Text c="navy.6" lh={1.65}>
              Free shipping on everything, a straightforward replacement route when something
              arrives damaged, and plain language about what handloom cloth does naturally. Every
              claim below is stated in full on the policy pages.
            </Text>
          </Stack>
        </Container>
      </Box>

      <Container size="xl" py="xl">
        <Flex gap="lg" align="flex-start" direction={{ base: 'column', lg: 'row' }}>
          <Box w={{ base: '100%', lg: 280 }} flex="none">
            <PoliciesNav />
          </Box>

          <Box flex={1} w="100%">
            <Stack gap="lg">
              <Card shadow="sm" radius="lg" padding="lg" withBorder={false} bg="brand.1">
                <Stack gap="sm">
                  <Text fz={11} fw={700} c="brand.6" style={{ letterSpacing: '0.12em' }}>
                    THE SHORT VERSION
                  </Text>
                  <Title
                    order={2}
                    fz={{ base: '1.5rem', sm: '1.75rem' }}
                    fw={600}
                    c="navy.9"
                    style={{ fontFamily: displayFont.style.fontFamily }}
                  >
                    Replacement for damage or defects
                  </Title>
                  <Text c="navy.7" lh={1.7}>
                    If a piece reaches you damaged in transit or with a manufacturing defect, we
                    replace it free of charge. Send us a continuous unboxing video within{' '}
                    {DAMAGE_CLAIM_WINDOW_HOURS} hours and we take it from there. We do not offer
                    returns or refunds for a change of mind.
                  </Text>
                  <Group gap="xs" mt={4}>
                    <Pill>Free replacement when approved</Pill>
                    <Pill>No cost to you</Pill>
                    <Pill>Refund if out of stock</Pill>
                  </Group>
                </Stack>
              </Card>

              <Box>
                <SectionHead eyebrow="Step by step" title="How a replacement works" />
                <SimpleGrid cols={{ base: 1, md: 3 }} spacing="md">
                  {STEPS.map(({ n, icon: Icon, title, body, foot }) => (
                    <Card key={n} shadow="sm" radius="lg" padding="lg" withBorder={false} h="100%">
                      <Stack gap="sm" h="100%">
                        <Group justify="space-between" align="center">
                          <Box
                            w={28}
                            h={28}
                            bg="brand.5"
                            style={{ borderRadius: 999, display: 'grid', placeItems: 'center' }}
                          >
                            <Text fz={12} fw={700} c="white">
                              {n}
                            </Text>
                          </Box>
                          <Icon width={18} height={18} color="var(--mantine-color-navy-5)" />
                        </Group>
                        <Text
                          fz="1.125rem"
                          fw={600}
                          c="navy.9"
                          style={{ fontFamily: displayFont.style.fontFamily }}
                        >
                          {title}
                        </Text>
                        <Text fz="xs" c="navy.6" lh={1.65}>
                          {body}
                        </Text>
                        <Text fz={10} fw={700} c="brand.5" mt="auto" style={{ letterSpacing: '0.06em' }}>
                          {foot.toUpperCase()}
                        </Text>
                      </Stack>
                    </Card>
                  ))}
                </SimpleGrid>
              </Box>

              <Card shadow="sm" radius="lg" padding="lg" withBorder={false} bg="navy.1">
                <Stack gap="md">
                  <Group gap={8} wrap="nowrap" align="center">
                    <SwatchIcon width={20} height={20} color="var(--mantine-color-brand-6)" />
                    <Title
                      order={2}
                      fz={{ base: '1.25rem', sm: '1.5rem' }}
                      fw={600}
                      c="navy.9"
                      style={{ fontFamily: displayFont.style.fontFamily }}
                    >
                      What handloom does naturally
                    </Title>
                  </Group>
                  <Text fz="sm" c="navy.7" lh={1.7}>
                    Handloom cloth is not machine output, and the policy says so: minor variations
                    in colour, weave, texture and dimension are inherent to it and are not treated
                    as defects.
                  </Text>

                  <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="sm">
                    {VARIATIONS.map(({ title, body }) => (
                      <Card key={title} radius="md" padding="md" withBorder={false}>
                        <Text
                          fz="1rem"
                          fw={600}
                          c="brand.6"
                          mb={4}
                          style={{ fontFamily: displayFont.style.fontFamily }}
                        >
                          {title}
                        </Text>
                        <Text fz="xs" c="navy.6" lh={1.6}>
                          {body}
                        </Text>
                      </Card>
                    ))}
                  </SimpleGrid>

                  <Group
                    gap="sm"
                    wrap="nowrap"
                    align="flex-start"
                    p="sm"
                    style={{ background: '#C7ECCE', borderRadius: 'var(--mantine-radius-md)' }}
                  >
                    <InformationCircleIcon
                      width={17}
                      height={17}
                      color="#2E4E37"
                      style={{ flexShrink: 0, marginTop: 2 }}
                    />
                    <Text fz="xs" c="#2E4E37" lh={1.6}>
                      Genuinely damaged or defective pieces are always eligible — this section is
                      about natural character, not faults.
                    </Text>
                  </Group>
                </Stack>
              </Card>

              <Box>
                <SectionHead eyebrow="Please note" title="What we cannot take back" />
                <Stack gap="sm">
                  {NOT_COVERED.map(({ title, body }) => (
                    <Group key={title} gap="sm" wrap="nowrap" align="flex-start">
                      <ExclamationTriangleIcon
                        width={17}
                        height={17}
                        color="var(--mantine-color-brand-5)"
                        style={{ flexShrink: 0, marginTop: 3 }}
                      />
                      <Box>
                        <Text fz="sm" fw={600} c="navy.9">
                          {title}
                        </Text>
                        <Text fz="xs" c="navy.6" lh={1.6}>
                          {body}
                        </Text>
                      </Box>
                    </Group>
                  ))}
                </Stack>
              </Box>

              <Card shadow="sm" radius="lg" padding="lg" withBorder={false} bg="navy.1">
                <SimpleGrid cols={{ base: 2, md: 4 }} spacing="md">
                  {ASSURANCES.map(({ icon: Icon, title, body }) => (
                    <Stack key={title} gap={4} align="center" ta="center">
                      <Icon width={22} height={22} color="var(--mantine-color-brand-6)" />
                      <Text fz="sm" fw={600} c="navy.9">
                        {title}
                      </Text>
                      <Text fz={10} c="navy.6" lh={1.5}>
                        {body}
                      </Text>
                    </Stack>
                  ))}
                </SimpleGrid>
              </Card>

              <Text fz="xs" c="navy.5" ta="center">
                Orders are dispatched in {DISPATCH_DAYS} business days and delivered in{' '}
                {DELIVERY_DAYS}. The policy pages linked on the left are the full and binding
                versions.
              </Text>
            </Stack>
          </Box>
        </Flex>
      </Container>
    </>
  );
}

function SectionHead({ eyebrow, title }: { eyebrow: string; title: string }) {
  return (
    <Stack gap={4} mb="md">
      <Text fz={11} fw={700} c="brand.5" style={{ letterSpacing: '0.14em' }}>
        {eyebrow.toUpperCase()}
      </Text>
      <Title
        order={2}
        fz={{ base: '1.25rem', sm: '1.5rem' }}
        fw={600}
        c="navy.9"
        style={{ fontFamily: displayFont.style.fontFamily }}
      >
        {title}
      </Title>
    </Stack>
  );
}

function Pill({ children }: { children: React.ReactNode }) {
  return (
    <Group gap={6} wrap="nowrap" px={10} py={5} bg="white" style={{ borderRadius: 999 }}>
      <CheckCircleIcon width={13} height={13} color="var(--mantine-color-brand-6)" />
      <Text fz={11} fw={600} c="navy.8">
        {children}
      </Text>
    </Group>
  );
}
