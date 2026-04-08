import type { Metadata } from 'next';
import { DM_Sans, Playfair_Display } from 'next/font/google';

import Footer from '@/components/layout/Footer';
import Header from '@/components/layout/Header';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';

import './globals.css';
import { Providers } from './providers';

const dmSans = DM_Sans({
  variable: '--font-sans',
  subsets: ['latin'],
  weight: ['400', '500', '600', '700'],
  display: 'swap',
});

const playfair = Playfair_Display({
  variable: '--font-heading',
  subsets: ['latin'],
  weight: ['400', '600', '700'],
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'Homechrome | Handloom Textiles',
  description:
    'Premium handloom textiles from across India. Sarees, dupattas, fabrics, and more.',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className={`${dmSans.variable} ${playfair.variable} font-sans antialiased`}>
        <TooltipProvider delay={300}>
          <Providers>
            <Header />
            <main className="min-h-screen">{children}</main>
            <Footer />
            <Toaster />
          </Providers>
        </TooltipProvider>
      </body>
    </html>
  );
}
