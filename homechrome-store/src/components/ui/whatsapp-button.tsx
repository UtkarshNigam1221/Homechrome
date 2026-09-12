'use client';

import { Anchor, Group, Text } from '@mantine/core';

import { WhatsAppIcon } from '@/components/ui/whatsapp-icon';
import { whatsappHref } from '@/lib/whatsapp';

interface WhatsAppButtonProps {
  /** Prefills the chat. */
  message: string;
  label: string;
  fullWidth?: boolean;
  size?: 'sm' | 'md';
  onClick?: () => void;
}

/** The design's green concierge action, used across the footer, drawer and pages. */
export function WhatsAppButton({
  message,
  label,
  fullWidth = true,
  size = 'md',
  onClick,
}: WhatsAppButtonProps) {
  return (
    <Anchor
      href={whatsappHref(message)}
      target="_blank"
      rel="noopener noreferrer"
      underline="never"
      onClick={onClick}
      w={fullWidth ? '100%' : undefined}
    >
      <Group
        gap={8}
        justify="center"
        wrap="nowrap"
        c="white"
        px={size === 'sm' ? 12 : 18}
        py={size === 'sm' ? 8 : 10}
        style={{
          background: 'var(--mantine-color-leaf-5)',
          borderRadius: 'var(--mantine-radius-md)',
        }}
      >
        <WhatsAppIcon size={size === 'sm' ? 15 : 17} />
        <Text fz={size === 'sm' ? 'xs' : 'sm'} fw={600} c="white">
          {label}
        </Text>
      </Group>
    </Anchor>
  );
}
