'use client';

import { Stepper } from '@mantine/core';

import { CheckoutStep } from './checkoutReducer';

const STEPS: { id: CheckoutStep; label: string }[] = [
  { id: 'address', label: 'Address' },
  { id: 'shipping', label: 'Shipping' },
  { id: 'review', label: 'Review' },
];

interface CheckoutProgressProps {
  step: CheckoutStep;
  onStepClick: (step: CheckoutStep) => void;
}

export function CheckoutProgress({ step, onStepClick }: CheckoutProgressProps) {
  const activeIndex = STEPS.findIndex((s) => s.id === step);

  return (
    <Stepper
      active={activeIndex}
      onStepClick={(idx) => onStepClick(STEPS[idx].id)}
      color="brand"
      mb="lg"
      size="sm"
      allowNextStepsSelect={false}
    >
      {STEPS.map((s) => (
        <Stepper.Step key={s.id} label={s.label} />
      ))}
    </Stepper>
  );
}
