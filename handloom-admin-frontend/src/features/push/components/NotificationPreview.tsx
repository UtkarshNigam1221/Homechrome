import { ChevronDown, Globe } from 'lucide-react';
import { useState } from 'react';

import type { PushAction } from '../types';

type PreviewMode = 'collapsed' | 'expanded' | 'lock';

interface NotificationPreviewProps {
  title: string;
  body: string;
  image?: string;
  /** Product thumbnail in place of the brand mark. */
  icon?: string;
  actions?: PushAction[];
  origin?: string;
}

const MODES: { id: PreviewMode; label: string; caption: string }[] = [
  {
    id: 'collapsed',
    label: 'Collapsed',
    caption: 'As it first arrives. Title and message are cut to one line each.',
  },
  {
    id: 'expanded',
    label: 'Expanded',
    caption: 'After the person pulls it down. Banner and buttons appear here.',
  },
  {
    id: 'lock',
    label: 'Lock screen',
    caption: 'Before unlocking. The message is hidden when sensitive content is off.',
  },
];

/**
 * A faithful Android notification, not a styled card.
 *
 * Operators kept asking why the logo sits on the right and an outline mark on
 * the left — that is Android's own row anatomy, so this reproduces it rather
 * than laying the assets out the way they read best.
 */
export function NotificationPreview({
  title,
  body,
  image,
  icon,
  actions = [],
  origin = 'homechrome.in',
}: NotificationPreviewProps) {
  const [mode, setMode] = useState<PreviewMode>('expanded');
  const [imageBroken, setImageBroken] = useState(false);
  const [iconBroken, setIconBroken] = useState(false);

  const shownTitle = title.trim() || 'Your title appears here';
  const shownBody = body.trim() || 'Your message appears here.';
  const now = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

  const expanded = mode === 'expanded';
  const locked = mode === 'lock';
  const shownActions = actions.filter((a) => a.title.trim()).slice(0, 2);
  const caption = MODES.find((m) => m.id === mode)?.caption;

  return (
    <div>
      <div className="mb-3 flex items-center justify-between gap-3">
        <h2 className="font-semibold text-gray-900">On the device</h2>
        <div className="flex rounded-lg bg-gray-100 p-0.5" role="tablist">
          {MODES.map((m) => (
            <button
              key={m.id}
              type="button"
              role="tab"
              aria-selected={mode === m.id}
              onClick={() => setMode(m.id)}
              className={`rounded-md px-2 py-1 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 ${
                mode === m.id
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              {m.label}
            </button>
          ))}
        </div>
      </div>

      {/* Wallpaper, so the shade reads as a phone rather than another panel. */}
      <div className="rounded-2xl bg-gradient-to-br from-slate-700 via-slate-800 to-indigo-950 p-3 sm:p-4">
        {locked && (
          <div className="pb-3 pt-1 text-center text-white/90">
            <p className="text-3xl font-light tracking-tight">{now}</p>
          </div>
        )}

        <div className="overflow-hidden rounded-[1.375rem] bg-[#1b1b1f] text-[#e3e3e3] shadow-lg">
          <div className="px-4 pt-3">
            <div className="flex items-center gap-1.5 text-[11px] leading-none text-[#a8a8ab]">
              {/* The monochrome badge sits here, at status-bar size. */}
              <Globe className="h-3.5 w-3.5 shrink-0" aria-hidden />
              <span className="truncate">{origin}</span>
              <span aria-hidden>•</span>
              <span className="shrink-0">now</span>
              <ChevronDown
                className={`ml-auto h-4 w-4 shrink-0 transition-transform ${
                  expanded ? 'rotate-180' : ''
                }`}
                aria-hidden
              />
            </div>

            <div className="mt-1.5 flex items-start gap-3 pb-3">
              <div className="min-w-0 flex-1">
                <p className={`text-sm font-medium text-white ${expanded ? '' : 'truncate'}`}>
                  {shownTitle}
                </p>
                {locked ? (
                  <p className="mt-0.5 text-sm italic text-[#8e8e91]">Contents hidden</p>
                ) : (
                  <p
                    className={`mt-0.5 text-sm leading-snug text-[#c7c7ca] ${
                      expanded ? 'whitespace-pre-wrap' : 'truncate'
                    }`}
                  >
                    {shownBody}
                  </p>
                )}
              </div>

              {/* Android pins the icon to the right. Not positionable. */}
              {!iconBroken && (
                <img
                  src={icon?.trim() || '/icon-preview.png'}
                  alt=""
                  aria-hidden
                  className="mt-0.5 h-10 w-10 shrink-0 rounded-full object-cover"
                  onError={() => setIconBroken(true)}
                />
              )}
            </div>
          </div>

          {expanded && image && !imageBroken && (
            <img
              src={image}
              alt=""
              className="h-36 w-full object-cover"
              onError={() => setImageBroken(true)}
            />
          )}

          {expanded && shownActions.length > 0 && (
            <div className="flex divide-x divide-white/10 border-t border-white/10">
              {shownActions.map((action) => (
                <span
                  key={action.action}
                  className="flex-1 truncate px-3 py-2.5 text-center text-xs font-semibold uppercase tracking-wide text-[#a8c7fa]"
                >
                  {action.title}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>

      {expanded && image && imageBroken && (
        <p className="mt-2 text-sm text-amber-700">
          That banner URL did not load. Devices will show the notification without it.
        </p>
      )}

      <p className="mt-3 text-sm text-gray-500">{caption}</p>
    </div>
  );
}
