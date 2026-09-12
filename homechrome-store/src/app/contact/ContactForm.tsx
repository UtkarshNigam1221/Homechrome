'use client';

import { PaperAirplaneIcon } from '@heroicons/react/24/outline';
import {
  Box,
  Button,
  Checkbox,
  Group,
  Select,
  SimpleGrid,
  Stack,
  Text,
  Textarea,
  TextInput,
  Title,
} from '@mantine/core';
import { useState } from 'react';

import { SUPPORT_WHATSAPP } from '@/lib/constants';

import { displayFont } from '../fonts';

const PURPOSES = [
  'Where is my order',
  'Question about a product',
  'Damaged or defective item',
  'Custom dimensions',
  'Something else',
];

/**
 * The design draws a submit form, but no enquiry endpoint exists. Rather than
 * ship a control that silently drops what people type, the fields compose a
 * WhatsApp message — the same details reach the same desk, and nothing is lost
 * if a send fails. Swap in a POST here once there is somewhere to post to.
 */
export function ContactForm() {
  const [name, setName] = useState('');
  const [phone, setPhone] = useState('');
  const [email, setEmail] = useState('');
  const [orderId, setOrderId] = useState('');
  const [purpose, setPurpose] = useState<string | null>(null);
  const [message, setMessage] = useState('');
  const [touched, setTouched] = useState(false);

  const missing = !name.trim() || !message.trim() || !purpose;

  const handleSend = () => {
    setTouched(true);
    if (missing) return;

    const lines = [
      `Hi Homechrome — ${purpose}`,
      '',
      `Name: ${name.trim()}`,
      phone.trim() && `Phone: ${phone.trim()}`,
      email.trim() && `Email: ${email.trim()}`,
      orderId.trim() && `Order ID: ${orderId.trim()}`,
      '',
      message.trim(),
    ].filter(Boolean);

    window.open(
      `https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent(lines.join('\n'))}`,
      '_blank',
      'noopener,noreferrer',
    );
  };

  return (
    <Stack gap="md">
      <Stack gap={6}>
        <Text fz={11} fw={700} c="brand.5" style={{ letterSpacing: '0.14em' }}>
          SEND US A MESSAGE
        </Text>
        <Title
          order={2}
          fz={{ base: '1.5rem', sm: '1.875rem' }}
          fw={600}
          c="navy.9"
          lh={1.2}
          style={{ fontFamily: displayFont.style.fontFamily }}
        >
          Tell us what you need
        </Title>
        <Text fz="sm" c="navy.6" lh={1.6}>
          Fill this in and we will open WhatsApp with your details ready to send — so nothing gets
          lost and you keep a copy of the conversation.
        </Text>
      </Stack>

      <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
        <TextInput
          label="Full name"
          placeholder="e.g. Radhika Singhania"
          required
          value={name}
          onChange={(e) => setName(e.currentTarget.value)}
          error={touched && !name.trim() ? 'Please tell us your name' : null}
        />
        <TextInput
          label="Phone number"
          placeholder="98765 43210"
          value={phone}
          onChange={(e) => setPhone(e.currentTarget.value)}
          inputMode="tel"
        />
        <TextInput
          label="Email address"
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.currentTarget.value)}
          inputMode="email"
        />
        <TextInput
          label="Order ID"
          description="Optional"
          placeholder="e.g. HC-2026-8841"
          value={orderId}
          onChange={(e) => setOrderId(e.currentTarget.value)}
        />
      </SimpleGrid>

      <Select
        label="What is this about"
        placeholder="Choose one"
        required
        data={PURPOSES}
        value={purpose}
        onChange={setPurpose}
        error={touched && !purpose ? 'Pick the closest option' : null}
      />

      <Textarea
        label="Your message"
        placeholder="Describe what you need — a size, a weave, or an order already on its way."
        required
        minRows={5}
        autosize
        value={message}
        onChange={(e) => setMessage(e.currentTarget.value)}
        error={touched && !message.trim() ? 'Please add a little detail' : null}
      />

      <Checkbox
        defaultChecked
        readOnly
        color="brand"
        label="Replies come to your WhatsApp number, so you keep the thread."
      />

      <Group>
        <Button
          color="brand"
          size="md"
          radius="sm"
          onClick={handleSend}
          rightSection={<PaperAirplaneIcon width={16} height={16} />}
        >
          Send on WhatsApp
        </Button>
      </Group>

      <Box>
        <Text fz="xs" c="navy.6" lh={1.6}>
          Your details go straight to our support number. We do not add them to a marketing list.
        </Text>
      </Box>
    </Stack>
  );
}
