import { ImageIcon, LayoutGrid, LinkIcon } from 'lucide-react';
import { useState } from 'react';

import { Button, ImageUpload, Input } from '@/shared/components/ui';

import { ProductImagePicker } from './ProductImagePicker';

type Mode = 'upload' | 'url' | 'products';

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
  const [pickerOpen, setPickerOpen] = useState(false);

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <label className="text-sm font-medium text-gray-700">{label}</label>
        <div className="flex gap-1 rounded-lg bg-gray-100 p-0.5" role="tablist">
          {(
            [
              ['upload', 'Upload', ImageIcon],
              ['url', 'Paste URL', LinkIcon],
              ['products', 'From products', LayoutGrid],
            ] as const
          ).map(([id, text, Icon]) => (
            <button
              key={id}
              type="button"
              role="tab"
              aria-selected={mode === id}
              onClick={() => setMode(id)}
              className={`flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 ${
                mode === id
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700'
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
      ) : mode === 'url' ? (
        <Input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="https://cdn.homechrome.in/assets/image/banner.jpg"
          hint={hint}
        />
      ) : (
        <div>
          {value && (
            <img
              src={value}
              alt=""
              className="mb-2 h-24 w-24 rounded-lg border border-gray-200 object-cover"
            />
          )}
          <Button type="button" variant="secondary" size="sm" onClick={() => setPickerOpen(true)}>
            Browse product images
          </Button>
          {hint && <p className="mt-1 text-sm text-gray-500">{hint}</p>}
          <ProductImagePicker
            opened={pickerOpen}
            onClose={() => setPickerOpen(false)}
            onSelect={onChange}
          />
        </div>
      )}
    </div>
  );
}
