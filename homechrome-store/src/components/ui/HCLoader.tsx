'use client';

import { useEffect, useState } from 'react';
import dynamic from 'next/dynamic';

import { Center, Overlay, VisuallyHidden } from '@mantine/core';

import loaderAnimation from '@/assets/loader.json';

// Lottie player uses `window` internally — must not run on the server.
const Lottie = dynamic(() => import('lottie-react'), { ssr: false });

export type HCLoaderSize = 'sm' | 'md' | 'lg' | 'fullPage';

interface HCLoaderProps {
  size?: HCLoaderSize;
  label?: string;
  className?: string;
}

const pixelSize: Record<Exclude<HCLoaderSize, 'fullPage'>, number> = {
  sm: 24,
  md: 48,
  lg: 96,
};

// The animation is paused on a mid-frame (frame 75 of 150) when the user
// prefers reduced motion. Visual presence preserved; no movement.
const REDUCED_MOTION_FRAME = 75;

function getReducedMotion(): boolean {
  if (typeof window === 'undefined') return false;
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

function useReducedMotion(): boolean {
  const [reduced, setReduced] = useState<boolean>(getReducedMotion);
  useEffect(() => {
    if (typeof window === 'undefined') return;
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');
    const handler = (e: MediaQueryListEvent) => setReduced(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);
  return reduced;
}

export default function HCLoader({
  size = 'md',
  label = 'Loading',
  className,
}: HCLoaderProps) {
  const reduced = useReducedMotion();

  const dims = size === 'fullPage' ? pixelSize.lg : pixelSize[size];

  const player = (
    <Lottie
      animationData={loaderAnimation}
      loop={!reduced}
      autoplay={!reduced}
      initialSegment={reduced ? [REDUCED_MOTION_FRAME, REDUCED_MOTION_FRAME + 1] : undefined}
      style={{ width: dims, height: dims }}
      aria-hidden="true"
    />
  );

  if (size === 'fullPage') {
    return (
      <Overlay
        fixed
        color="#fff"
        backgroundOpacity={0.8}
        blur={4}
        // 400 = Mantine's `overlay` elevation, above Modal/Drawer (200) so the
        // full-page loader always wins over any open modal stack.
        zIndex={400}
        className={className}
      >
        <Center h="100%" role="status" aria-live="polite">
          {player}
          <VisuallyHidden>{label}</VisuallyHidden>
        </Center>
      </Overlay>
    );
  }

  return (
    <Center inline role="status" aria-live="polite" className={className}>
      {player}
      <VisuallyHidden>{label}</VisuallyHidden>
    </Center>
  );
}
