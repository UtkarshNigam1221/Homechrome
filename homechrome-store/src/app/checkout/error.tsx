'use client';

import { Button } from '@/components/ui/button';
import { Container } from '@/components/ui/container';

export default function CheckoutError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <Container className="flex min-h-[60vh] flex-col items-center justify-center py-16 text-center">
      <h2 className="text-2xl font-semibold text-foreground">Checkout Error</h2>
      <p className="mt-3 max-w-md text-base text-muted-foreground">
        {error.message || 'Something went wrong during checkout. Your cart is still saved.'}
      </p>
      <div className="mt-8 flex gap-4">
        <Button onClick={reset}>Try again</Button>
        <Button variant="outline" asChild>
          <a href="/cart">Back to Cart</a>
        </Button>
      </div>
    </Container>
  );
}
