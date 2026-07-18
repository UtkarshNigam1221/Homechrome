'use client';

import { Alert, Anchor, Box, Container, SimpleGrid, Stack } from '@mantine/core';
import Link from 'next/link';

import OrderSummary from '@/components/checkout/OrderSummary';
import CheckoutSkeleton from '@/components/skeleton/CheckoutSkeleton';
import { PageHeader } from '@/components/ui/page-header';
import { useAuthStore } from '@/stores/auth';

import { AddressStep } from './AddressStep';
import { CheckoutProgress } from './CheckoutProgress';
import { ReviewStep } from './ReviewStep';
import { useCheckoutState } from './useCheckoutState';

export default function CheckoutPage() {
  const authLoading = useAuthStore((s) => s.isLoading);
  const {
    state,
    dispatch,
    addresses,
    selectedAddress,
    handleAddressNext,
    handleAddAddress,
    handlePayNow,
    goToStep,
    creatingAddress,
    initiatingCheckout,
  } = useCheckoutState();

  const {
    cart,
    cartLoading,
    step,
    selectedAddressId,
    showAddressForm,
    addressSaving,
    initiating,
    error,
  } = state;

  if (authLoading || cartLoading) {
    return <CheckoutSkeleton />;
  }

  if (!cart || !cart.items || cart.items.length === 0) {
    return null;
  }

  return (
    <Container size="lg" py="lg">
      <PageHeader
        title="Checkout"
        description="Complete your order in a few simple steps."
      />

      <CheckoutProgress step={step} onStepClick={goToStep} />

      {error && <Alert color="red" mb="md">{error}</Alert>}

      <SimpleGrid cols={{ base: 1, lg: 3 }} spacing="xl">
        <Stack gap="md" style={{ gridColumn: 'span 2 / span 2' }}>
          {step === 'address' && (
            <AddressStep
              addresses={addresses}
              selectedAddressId={selectedAddressId}
              showAddressForm={showAddressForm}
              addressSaving={addressSaving}
              creatingAddress={creatingAddress}
              onSelectAddress={(id) => dispatch({ type: 'SELECT_ADDRESS', id })}
              onToggleForm={(show) => dispatch({ type: 'TOGGLE_ADDRESS_FORM', show })}
              onSaveAddress={handleAddAddress}
              onContinue={handleAddressNext}
            />
          )}

          {step === 'review' && (
            <ReviewStep
              selectedAddress={selectedAddress}
              items={cart.items}
              initiating={initiating}
              initiatingCheckout={initiatingCheckout}
              onChangeAddress={() => dispatch({ type: 'GO_TO_STEP', step: 'address' })}
              onPayNow={handlePayNow}
            />
          )}
        </Stack>

        <Box>
          <Box pos="sticky" top={144}>
            <OrderSummary
              items={cart.items}
              subtotal={cart.cart.subtotal}
            />
            <Box ta="center" mt="md">
              <Anchor component={Link} href="/cart" size="sm" c="brand">
                Back to Cart
              </Anchor>
            </Box>
          </Box>
        </Box>
      </SimpleGrid>
    </Container>
  );
}
