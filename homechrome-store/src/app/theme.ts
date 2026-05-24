import { createTheme, MantineColorsTuple } from '@mantine/core';

import { siteFont } from './fonts';

// Brand gold — matches --primary (#D4A574) and --primary-dark (#B8894E)
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

// Navy — matches --foreground (#1C2951)
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
  },
  defaultRadius: 'md',
  cursorType: 'pointer',
});
