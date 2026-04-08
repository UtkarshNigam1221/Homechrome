'use client';

import { Button } from '@/components/ui/button';
import { Container } from '@/components/ui/container';

export default function Error({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <Container className="flex min-h-[40vh] flex-col items-center justify-center py-16 text-center">
      <h2 className="text-2xl font-semibold text-foreground">Something went wrong</h2>
      <p className="mt-3 max-w-md text-base text-muted-foreground">
        {error.message || 'We could not load your account information. Please try again.'}
      </p>
      <Button className="mt-8" onClick={reset}>
        Try again
      </Button>
    </Container>
  );
}
