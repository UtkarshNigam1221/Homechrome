import { ColorSchemeScript, mantineHtmlProps } from '@mantine/core';
import type { Metadata } from 'next';

import { MiniCartDrawer } from '@/components/cart/MiniCartDrawer';
import EmbedderWarmer from '@/components/EmbedderWarmer';
import { FloatingActions } from '@/components/layout/FloatingActions';
import Footer from '@/components/layout/Footer';
import Header from '@/components/layout/Header';
import { SpotlightSearch } from '@/components/search/SpotlightSearch';
import { API_BASE } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Category } from '@/types';

import { siteFont } from './fonts';
import './globals.css';
import { Providers } from './providers';

async function getCategories(): Promise<Category[]> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.CATEGORIES}`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}

export const metadata: Metadata = {
  title: 'Homechrome | Handloom Textiles',
  description:
    'Premium handloom textiles from across India. Sarees, dupattas, fabrics, and more.',
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const categories = await getCategories();
  return (
    <html lang="en" {...mantineHtmlProps}>
      <head>
        <ColorSchemeScript defaultColorScheme="light" />
      </head>
      <body className={siteFont.className} style={{ minHeight: '100vh' }}>
        <Providers>
          <EmbedderWarmer />
          <Header categories={categories} />
          <SpotlightSearch categories={categories} />
          <MiniCartDrawer />
          <FloatingActions />
          <main style={{ minHeight: '100vh' }}>{children}</main>
          <Footer />
        </Providers>
      </body>
    </html>
  );
}
