import { Button, createTheme, MantineColorsTuple } from '@mantine/core';

import { siteFont } from './fonts';

// The original Homechrome tan ramp, kept through the Stitch layout rebuild.
const brand: MantineColorsTuple = [
  '#FBF7F2',
  '#F4E9DC',
  '#ECDAC3',
  '#E2C7A4',
  '#DBB78B',
  '#D4A574',
  '#B8894E',
  '#946B3A',
  '#6F4F2A',
  '#4B341B',
];

// Shades 0–3 are the warm surfaces the Stitch components lean on (page ground,
// panels, borders); 4–9 are the original navy ink. One ramp, two jobs.
const navy: MantineColorsTuple = [
  '#F5F0EB',
  '#EFE8E0',
  '#E6DDD2',
  '#D9CEC0',
  '#5C6A8E',
  '#424F75',
  '#2D3B62',
  '#1C2951',
  '#131B3A',
  '#0A0E22',
];

// Green and blue-grey accent families used by status pills, WhatsApp actions
// and the steep-discount badge.
const leaf: MantineColorsTuple = [
  '#EDF7EF',
  '#C7ECCE',
  '#ABCFB2',
  '#8DB697',
  '#5B7C63',
  '#42634C',
  '#2E4E37',
  '#1F3B27',
  '#132C19',
  '#01210F',
];

const mist: MantineColorsTuple = [
  '#F2F5FB',
  '#DBE3F3',
  '#BFC7D7',
  '#9BA6BC',
  '#75819A',
  '#575F6D',
  '#3F4754',
  '#2D3441',
  '#1F2531',
  '#141C28',
];

export const theme = createTheme({
  primaryColor: 'brand',
  primaryShade: 5,
  black: '#1C2951',
  white: '#FFFFFF',
  colors: { brand, navy, leaf, mist },
  fontFamily: siteFont.style.fontFamily,
  fontFamilyMonospace: 'ui-monospace, SFMono-Regular, monospace',
  headings: {
    fontFamily: siteFont.style.fontFamily,
    fontWeight: '600',
    sizes: {
      h1: { fontSize: '2.5rem', lineHeight: '1.15', fontWeight: '700' },
      h2: { fontSize: '2rem', lineHeight: '1.2', fontWeight: '700' },
      h3: { fontSize: '1.5rem', lineHeight: '1.3', fontWeight: '600' },
    },
  },
  defaultRadius: 'lg',
  cursorType: 'pointer',
  shadows: {
    xs: '0 1px 2px rgba(28,41,81,0.04)',
    sm: '0 2px 6px rgba(28,41,81,0.06)',
    md: '0 6px 16px rgba(28,41,81,0.08)',
    lg: '0 12px 28px rgba(28,41,81,0.10)',
    xl: '0 20px 48px rgba(28,41,81,0.12)',
  },
  components: {
    Button: Button.extend({
      defaultProps: {
        radius: 'xl',
      },
    }),
  },
});
