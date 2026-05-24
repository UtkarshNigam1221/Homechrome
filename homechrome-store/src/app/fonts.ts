// Single source of truth for site typography.
// To change the site-wide font: swap the import below + weights, and rerun.
// Mantine theme + plain HTML body consume `siteFont.style.fontFamily`.

import { Roboto } from 'next/font/google';

export const siteFont = Roboto({
  subsets: ['latin'],
  weight: ['400', '500', '700'],
  display: 'swap',
});
