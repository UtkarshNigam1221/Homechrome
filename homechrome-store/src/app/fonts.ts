// Single source of truth for site typography.
// To change the site-wide font: swap the import below + weights, and rerun.
// Mantine theme + plain HTML body consume `siteFont.style.fontFamily`.

import { Playfair_Display, Roboto } from 'next/font/google';

export const siteFont = Roboto({
  subsets: ['latin', 'latin-ext'],
  weight: ['400', '500', '700'],
  display: 'swap',
});

// Display serif — used on hero / headlines that need editorial feel.
export const displayFont = Playfair_Display({
  subsets: ['latin'],
  weight: ['700'],
  display: 'swap',
});
