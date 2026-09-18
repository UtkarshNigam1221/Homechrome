import { ChevronDown } from 'lucide-react';
import { useState } from 'react';

interface NotificationPreviewProps {
  title: string;
  body: string;
  image?: string;
  /** Storefront origin, shown in the notification header the way Chrome does. */
  origin?: string;
}

/**
 * A faithful Android notification shade, not a styled card.
 *
 * Operators kept asking why the logo appears on the right and an outline mark
 * on the left — that is Android's own row anatomy, so the preview reproduces it
 * exactly rather than laying the assets out the way they read best.
 */
export function NotificationPreview({
  title,
  body,
  image,
  origin = 'dev-store.homechrome.in',
}: NotificationPreviewProps) {
  const [expanded, setExpanded] = useState(true);
  const [imageBroken, setImageBroken] = useState(false);

  const shownTitle = title.trim() || 'Your title appears here';
  const shownBody = body.trim() || 'Your message appears here.';
  const now = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

  return (
    <div>
      <div className="mb-3 flex items-baseline justify-between gap-3">
        <h2 className="font-semibold text-gray-900">On the device</h2>
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          className="rounded text-sm text-primary-700 underline-offset-2 hover:underline focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          {expanded ? 'Show collapsed' : 'Show expanded'}
        </button>
      </div>

      {/* Wallpaper, so the shade reads as a phone rather than another panel. */}
      <div className="rounded-2xl bg-gradient-to-br from-slate-700 via-slate-800 to-indigo-950 p-3 sm:p-4">
        <div className="overflow-hidden rounded-[1.375rem] bg-[#1b1b1f] text-[#e3e3e3] shadow-lg">
          <div className="px-4 pt-3">
            {/* Header row: the monochrome badge lives here, at status-bar size. */}
            <div className="flex items-center gap-1.5 text-[11px] leading-none text-[#a8a8ab]">
              <img
                src="/badge-preview.png"
                alt=""
                aria-hidden
                className="h-3.5 w-3.5 opacity-90"
                onError={(e) => {
                  e.currentTarget.style.display = 'none';
                }}
              />
              {/* Android prefixes the browser name; dropped here because at this
                  width it truncates both halves to nothing useful. */}
              <span className="truncate">{origin}</span>
              <span aria-hidden>•</span>
              <span className="shrink-0">{now}</span>
              <ChevronDown
                className={`ml-auto h-4 w-4 shrink-0 transition-transform ${
                  expanded ? 'rotate-180' : ''
                }`}
                aria-hidden
              />
            </div>

            <div className="mt-1.5 flex items-start gap-3 pb-3">
              <div className="min-w-0 flex-1">
                <p
                  className={`text-sm font-medium text-white ${expanded ? '' : 'truncate'}`}
                  title={shownTitle}
                >
                  {shownTitle}
                </p>
                <p
                  className={`mt-0.5 text-sm leading-snug text-[#c7c7ca] ${
                    expanded ? 'whitespace-pre-wrap' : 'truncate'
                  }`}
                >
                  {shownBody}
                </p>
              </div>

              {/* Android pins the colour icon to the right. Not positionable. */}
              <img
                src="/icon-preview.png"
                alt=""
                aria-hidden
                className="mt-0.5 h-10 w-10 shrink-0 rounded-lg"
                onError={(e) => {
                  e.currentTarget.style.display = 'none';
                }}
              />
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
        </div>
      </div>

      {expanded && image && imageBroken && (
        <p className="mt-2 text-sm text-amber-700">
          That banner URL did not load. Devices will show the notification without it.
        </p>
      )}

      <p className="mt-3 text-sm text-gray-500">
        {expanded
          ? 'Expanded, after the person pulls the notification down.'
          : 'Collapsed, as it first arrives. Title and message are cut to one line each.'}
      </p>
    </div>
  );
}
