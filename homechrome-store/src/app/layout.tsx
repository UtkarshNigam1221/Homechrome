import type { Metadata } from 'next';
import { Inter } from 'next/font/google';

import Footer from '@/components/layout/Footer';
import Header from '@/components/layout/Header';

import './globals.css';
import { Providers } from './providers';

const inter = Inter({
  variable: '--font-inter',
  subsets: ['latin'],
});

export const metadata: Metadata = {
  title: 'Homechrome | Handloom Textiles',
  description:
    'Premium handloom textiles from master artisans across India. Sarees, dupattas, fabrics, and more.',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={`${inter.variable} antialiased`}>
        <Providers>
          <Header />
          <main className="min-h-screen">{children}</main>
          <Footer />
        </Providers>
      </body>
    </html>
  );
}
