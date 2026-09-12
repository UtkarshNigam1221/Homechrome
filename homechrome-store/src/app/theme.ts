import { Button, createTheme, MantineColorsTuple } from '@mantine/core';

import { displayFont, siteFont } from './fonts';

// Terracotta ramp. Shades 1, 2, 4, 5, 6 and 9 are the design system's own
// tokens (primary-fixed, primary-container, primary, on-primary-fixed); rest interpolate.
const brand: MantineColorsTuple = [
  '#FFF1EC',
  '#FFDBD0',
  '#FFB59E',
  '#EE8A66',
  '#BA5838',
  '#9A4023',
  '#7E2C10',
  '#66230C',
  '#4E1804',
  '#3A0B00',
];

// Warm neutral ink ramp — the design's four surface steps at the low end,
// its outline and text at the high end. Still named `navy` for its 80 call
// sites; it is no longer blue.
const navy: MantineColorsTuple = [
  '#FCF9F4',
  '#F6F3EE',
  '#F0EDE9',
  '#E5E2DD',
  '#89726B',
  '#6D5A54',
  '#56423D',
  '#3D332F',
  '#31302D',
  '#1C1C19',
];

// The design system's tertiary (green) and secondary (blue-grey) families.
// Named steps come from its own tokens; the gaps interpolate.
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
  black: '#1C1C19',
  white: '#FFFFFF',
  colors: { brand, navy, leaf, mist },
  fontFamily: siteFont.style.fontFamily,
  fontFamilyMonospace: 'ui-monospace, SFMono-Regular, monospace',
  headings: {
    fontFamily: displayFont.style.fontFamily,
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
    xs: '0 1px 2px rgba(28,28,25,0.04)',
    sm: '0 2px 6px rgba(28,28,25,0.06)',
    md: '0 6px 16px rgba(28,28,25,0.08)',
    lg: '0 12px 28px rgba(28,28,25,0.10)',
    xl: '0 20px 48px rgba(28,28,25,0.12)',
  },
  components: {
    Button: Button.extend({
      defaultProps: {
        radius: 'xl',
      },
    }),
  },
});
