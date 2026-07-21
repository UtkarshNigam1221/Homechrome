'use client';

import { Anchor } from '@mantine/core';
import Link from 'next/link';

// Thin client wrapper so the (server-rendered, metadata-exporting) legal
// pages can still link to each other via Next's client-side router. Mantine's
// `component={Link}` polymorphic prop passes the Link component itself as a
// prop, which can't cross the server/client boundary from a Server Component
// — only this file needs 'use client' for that.
export function LegalLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <Anchor component={Link} href={href}>
      {children}
    </Anchor>
  );
}
