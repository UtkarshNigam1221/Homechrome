// Single source of truth for site typography.
// To change the site-wide font: swap the import below + weights, and rerun.
// Mantine theme + plain HTML body consume `siteFont.style.fontFamily`.

import { Playfair_Display, Plus_Jakarta_Sans } from 'next/font/google';

export const siteFont = Plus_Jakarta_Sans({
  subsets: ['latin', 'latin-ext'],
  weight: ['400', '500', '600', '700'],
  display: 'swap',
});

// Display serif — headings site-wide, plus hero / editorial callouts.
export const displayFont = Playfair_Display({
  subsets: ['latin'],
  weight: ['500', '600', '700'],
  display: 'swap',
});
