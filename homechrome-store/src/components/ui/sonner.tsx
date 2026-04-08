'use client';

import { Toaster as Sonner, type ToasterProps } from 'sonner';

function Toaster(props: ToasterProps) {
  return (
    <Sonner
      theme="light"
      className="toaster group"
      style={
        {
          '--normal-bg': 'var(--popover)',
          '--normal-text': 'var(--popover-foreground)',
          '--normal-border': 'var(--border)',
          '--border-radius': 'var(--radius)',
          '--success-bg': 'var(--primary)',
          '--success-text': 'white',
        } as React.CSSProperties
      }
      position="bottom-right"
      duration={3000}
      {...props}
    />
  );
}

export { Toaster };
