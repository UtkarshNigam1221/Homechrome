'use client';

import { useEffect, useState } from 'react';

// A prompt that appears after a delay and, once dismissed, stays away for a
// number of days. A blocked localStorage (private mode) falls back to showing it.
// `eligible` may be a function so browser-only checks run inside the effect.
export function useDelayedPrompt(
  storageKey: string,
  dismissDays: number,
  delayMs: number,
  eligible: boolean | (() => boolean),
) {
  const [shown, setShown] = useState(false);

  useEffect(() => {
    if (!(typeof eligible === 'function' ? eligible() : eligible)) return;
    try {
      const dismissedAt = Number(localStorage.getItem(storageKey));
      if (dismissedAt && (Date.now() - dismissedAt) / 86_400_000 < dismissDays) return;
    } catch {
      /* nothing readable; showing is the safe default */
    }
    const timer = setTimeout(() => setShown(true), delayMs);
    return () => clearTimeout(timer);
  }, [eligible, storageKey, dismissDays, delayMs]);

  const dismiss = () => {
    setShown(false);
    try {
      localStorage.setItem(storageKey, String(Date.now()));
    } catch {
      /* nothing to persist to; it returns next visit */
    }
  };

  return { shown, dismiss };
}
