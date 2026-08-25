'use client';

import { Button, Group, Pill, Stack, Text, TextInput } from '@mantine/core';
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

  // Applied state is a removable chip, not text sitting in an input.
  if (appliedCode) {
    return (
      <Pill
        size="lg"
        withRemoveButton
        onRemove={onRemoved}
        removeButtonProps={{ 'aria-label': `Remove coupon ${appliedCode}` }}
      >
        {appliedCode} · {formatPrice(appliedDiscount)} off
      </Pill>
    );
  }

  return (
    <Stack gap="xs">
      <Group gap="xs" wrap="nowrap">
        <TextInput
          placeholder="Coupon code"
          aria-label="Coupon code"
          value={code}
          onChange={(e) => setCode(e.currentTarget.value.toUpperCase())}
          onKeyDown={(e) => e.key === 'Enter' && apply()}
          error={!!error}
          style={{ flex: 1 }}
        />
        <Button onClick={apply} loading={checking} variant="light" color="brand">
          Apply
        </Button>
      </Group>
      {error && <Text size="xs" c="red.7">{error}</Text>}
    </Stack>
  );
}
