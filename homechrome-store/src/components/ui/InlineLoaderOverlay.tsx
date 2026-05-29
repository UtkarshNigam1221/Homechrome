'use client';

import { Center, Overlay } from '@mantine/core';

import HCLoader, { HCLoaderSize } from './HCLoader';

interface InlineLoaderOverlayProps {
  // Caller must wrap in an element with `position: relative` (Box pos="relative"
  // or equivalent). This component renders Mantine <Overlay> absolutely
  // positioned, which targets the nearest positioned ancestor.
  visible: boolean;
  size?: Exclude<HCLoaderSize, 'fullPage'>;
  label?: string;
  // zIndex defaults to 200 (= Mantine's `modal` elevation), placing the overlay
  // above ordinary content and equal-tier with portaled dropdowns inside the
  // same stacking context. Bump per call site only when fighting a specific
  // overlapping element.
  zIndex?: number;
}

export default function InlineLoaderOverlay({
  visible,
  size = 'sm',
  label = 'Loading',
  zIndex = 200,
}: InlineLoaderOverlayProps) {
  if (!visible) return null;
  return (
    <Overlay color="#fff" backgroundOpacity={0.7} blur={1} zIndex={zIndex}>
      <Center h="100%">
        <HCLoader size={size} label={label} />
      </Center>
    </Overlay>
  );
}
