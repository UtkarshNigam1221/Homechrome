import { ImageIcon, LinkIcon } from 'lucide-react';
import { useState } from 'react';

import { ImageUpload, Input } from '@/shared/components/ui';

type Mode = 'upload' | 'url';

interface BannerImageFieldProps {
  value: string;
  onChange: (value: string) => void;
  label: string;
  hint?: string;
}

/**
 * An operator has no way to know which asset URLs exist, so typing one is not a
 * real option. Upload is the default; pasting stays for externally hosted art.
 */
export function BannerImageField({ value, onChange, label, hint }: BannerImageFieldProps) {
  const [mode, setMode] = useState<Mode>('upload');

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <label className="text-sm font-medium text-gray-700">{label}</label>
        <div className="flex gap-1 rounded-lg bg-gray-100 p-0.5">
          {(
            [
              ['upload', 'Upload', ImageIcon],
              ['url', 'Paste URL', LinkIcon],
            ] as const
          ).map(([id, text, Icon]) => (
            <button
              key={id}
              type="button"
              onClick={() => setMode(id)}
              className={`flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium ${
                mode === id ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500'
              }`}
            >
              <Icon className="h-3 w-3" />
              {text}
            </button>
          ))}
        </div>
      </div>

      {mode === 'upload' ? (
        <ImageUpload
          value={value}
          onChange={(v) => onChange(Array.isArray(v) ? (v[0] ?? '') : v)}
          hint={hint}
          accept="image/*"
          maxSizeMB={1}
        />
      ) : (
        <Input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="https://cdn.homechrome.in/assets/image/banner.jpg"
          hint={hint}
        />
      )}
    </div>
  );
}
