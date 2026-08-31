'use client';

import { Button, Group, Stack, Text, TextInput, UnstyledButton } from '@mantine/core';
import { useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatPrice } from '@/lib/utils';
import { CouponValidationResult } from '@/types';

interface CouponInputProps {
  onApplied: (code: string, discount: number) => void;
  onRemoved: () => void;
  appliedCode?: string;
  appliedDiscount?: number;
}

export default function CouponInput({
  onApplied,
  onRemoved,
  appliedCode,
  appliedDiscount = 0,
}: CouponInputProps) {
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [checking, setChecking] = useState(false);

  const apply = async () => {
    const trimmed = code.trim();
    if (!trimmed) return;
    setChecking(true);
    setError('');
    try {
      // Advisory only. Checkout re-validates and the server wins — the two can
      // disagree if the cart changed since the code was entered.
      const { data } = await api.post<CouponValidationResult>(
        ROUTES.CHECKOUT.VALIDATE_COUPON,
        { code: trimmed },
      );
      if (data.valid) {
        onApplied(data.code, data.discount_amount ?? 0);
        setCode('');
      } else {
        // Server wording, verbatim when present — the copy lives in one place.
        setError(data.error_message || 'That code is not valid.');
      }
    } catch {
      setError("We couldn't check that code. Try again.");
    } finally {
      setChecking(false);
    }
  };

  // Applied: written as the deduction it is, in the same ledger column the offers use,
  // so the customer reads one figure in one place rather than a tag they must decode.
  if (appliedCode) {
    return (
      <Group
        gap="sm"
        wrap="nowrap"
        align="center"
        py={8}
        px={12}
        style={{
          background: 'var(--mantine-color-brand-1)',
          borderInlineStart: '3px solid var(--mantine-color-brand-5)',
          borderRadius: '0 var(--mantine-radius-sm) var(--mantine-radius-sm) 0',
        }}
      >
        <Stack gap={1} style={{ flex: 1, minWidth: 0 }}>
          <Text
            fz="xs"
            fw={700}
            c="navy.7"
            style={{
              fontFamily: 'var(--mantine-font-family-monospace)',
              letterSpacing: '0.06em',
            }}
          >
            {appliedCode}
          </Text>
          <Text fz={11} c="navy.4">
            applied to this order
          </Text>
        </Stack>
        <Text
          fz="sm"
          fw={700}
          c="navy.7"
          style={{ fontVariantNumeric: 'tabular-nums', minWidth: '4.5rem', textAlign: 'right' }}
        >
          −{formatPrice(appliedDiscount)}
        </Text>
        <UnstyledButton
          onClick={onRemoved}
          aria-label={`Remove coupon ${appliedCode}`}
          style={{ borderRadius: 'var(--mantine-radius-sm)' }}
        >
          <Text fz={11} c="navy.4" td="underline" px={2}>
            Remove
          </Text>
        </UnstyledButton>
      </Group>
    );
  }

  return (
    <Stack gap={6}>
      <Group gap="xs" wrap="nowrap" align="flex-start">
        <TextInput
          placeholder="Enter a code"
          aria-label="Coupon code"
          value={code}
          onChange={(e) => setCode(e.currentTarget.value.toUpperCase())}
          onKeyDown={(e) => e.key === 'Enter' && apply()}
          error={!!error}
          style={{ flex: 1 }}
          styles={{
            input: {
              fontFamily: 'var(--mantine-font-family-monospace)',
              letterSpacing: '0.08em',
            },
          }}
        />
        <Button onClick={apply} loading={checking} variant="light" color="brand">
          Apply
        </Button>
      </Group>
      {/* Announced, because a refusal is the whole answer to what the customer just did. */}
      {error && (
        <Text role="status" fz={11} c="red.7">
          {error}
        </Text>
      )}
    </Stack>
  );
}
