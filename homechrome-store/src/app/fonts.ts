// Single source of truth for site typography. Mantine and the body consume
// `siteFont`; `displayFont` is exposed as --font-display and applied to h1/h2
// from globals.css.

import { Playfair_Display, Roboto } from 'next/font/google';

export const siteFont = Roboto({
  subsets: ['latin', 'latin-ext'],
  weight: ['400', '500', '700'],
  display: 'swap',
});

export const displayFont = Playfair_Display({
  subsets: ['latin'],
  weight: ['600', '700'],
  display: 'swap',
  variable: '--font-display',
});
