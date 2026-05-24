'use client';

import { Button, Center, Container, Stack, Text, Title } from '@mantine/core';

export default function Error({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <Container size="md" py="xl">
      <Center mih="60vh">
        <Stack align="center" gap="md" ta="center">
          <Title order={2}>Something went wrong</Title>
          <Text c="dimmed" maw={480}>
            {error.message || 'An unexpected error occurred. Please try again.'}
          </Text>
          <Button mt="md" color="brand" onClick={reset}>
            Try again
          </Button>
        </Stack>
      </Center>
    </Container>
  );
}
