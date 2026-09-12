import { ColorSchemeScript, mantineHtmlProps } from '@mantine/core';
import type { Metadata } from 'next';

import { MiniCartDrawer } from '@/components/cart/MiniCartDrawer';
import OffersBanner from '@/components/catalog/OffersBanner';
import EmbedderWarmer from '@/components/EmbedderWarmer';
import Footer from '@/components/layout/Footer';
import Header from '@/components/layout/Header';
import { MobileTabBar } from '@/components/layout/MobileTabBar';
import PushOptInBanner from '@/components/notifications/PushOptInBanner';
import { SpotlightSearchLoader } from '@/components/search/SpotlightSearchLoader';
import { API_BASE, IS_INDEXABLE, SITE_URL } from '@/lib/constants';
import { ROUTES } from '@/lib/routes';
import { Category, PublicCoupon } from '@/types';

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

async function getPublicCoupons(): Promise<PublicCoupon[]> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.COUPONS}`, {
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
  // Absolute base for canonical + Open Graph URLs (fixes Next's metadataBase
  // warning). Derives from SITE_URL, so it tracks whatever canonical host is set.
  metadataBase: new URL(SITE_URL),
  // Non-prod hosts (dev, local) get noindex so they can't surface in search.
  ...(IS_INDEXABLE ? {} : { robots: { index: false, follow: false } }),
  title: 'Homechrome | Handloom Textiles',
  description:
    'Premium handloom textiles from across India. Sarees, dupattas, fabrics, and more.',
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const [categories, coupons] = await Promise.all([getCategories(), getPublicCoupons()]);
  return (
    <html lang="en" {...mantineHtmlProps}>
      <head>
        <ColorSchemeScript defaultColorScheme="light" />
      </head>
      <body className={siteFont.className} style={{ minHeight: '100vh' }}>
        <Providers>
          <EmbedderWarmer />
          <OffersBanner coupons={coupons} />
          <Header categories={categories} />
          <SpotlightSearchLoader categories={categories} />
          <MiniCartDrawer />
          <MobileTabBar />
          <PushOptInBanner />
          <main style={{ minHeight: '100vh' }}>{children}</main>
          <Footer categories={categories} />
        </Providers>
      </body>
    </html>
  );
}
