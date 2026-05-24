'use client';

import { Button, Center, Container, Group, Stack, Text, Title } from '@mantine/core';

export default function CheckoutError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <Container size="md" py="xl">
      <Center mih="60vh">
        <Stack align="center" gap="md" ta="center">
          <Title order={2}>Checkout Error</Title>
          <Text c="dimmed" maw={480}>
            {error.message || 'Something went wrong during checkout. Your cart is still saved.'}
          </Text>
          <Group mt="md" gap="sm">
            <Button color="brand" onClick={reset}>Try again</Button>
            <Button variant="outline" color="navy" onClick={() => (window.location.href = '/cart')}>
              Back to Cart
            </Button>
          </Group>
        </Stack>
      </Center>
    </Container>
  );
}
