import { Button, createTheme, MantineColorsTuple } from '@mantine/core';

import { siteFont } from './fonts';

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

const navy: MantineColorsTuple = [
  '#EBEDF4',
  '#C6CCDB',
  '#A0A9C0',
  '#7B85A4',
  '#5C6A8E',
  '#424F75',
  '#2D3B62',
  '#1C2951',
  '#131B3A',
  '#0A0E22',
];

export const theme = createTheme({
  primaryColor: 'brand',
  primaryShade: 5,
  black: '#1C2951',
  white: '#FFFFFF',
  colors: { brand, navy },
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
      // The brand tan (brand.5 #D4A574) is too light for white button text
      // (2.23:1, fails WCAG). For FILLED brand CTAs, use a deeper shade so the
      // default white text passes (white on brand.7 #946B3A = 5.24:1). Tan is
      // kept for accents; outline/subtle brand buttons are left untouched.
      vars: (_theme, props) => {
        const filledBrand =
          (!props.color || props.color === 'brand') &&
          (!props.variant || props.variant === 'filled');
        return filledBrand
          ? {
              root: {
                '--button-bg': 'var(--mantine-color-brand-7)',
                '--button-hover': 'var(--mantine-color-brand-8)',
              },
            }
          : { root: {} };
      },
    }),
  },
});
