import imageCompression from 'browser-image-compression';
import { clsx } from 'clsx';
import { Film, Image as ImageIcon, Loader2, Upload, X } from 'lucide-react';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import toast from 'react-hot-toast';

import { assetsApi } from '@/shared/api/assets';
import type { AssetType } from '@/shared/types/common';

export interface UploadedImage {
  id?: string;
  url: string;
  name: string;
}

interface ImageUploadProps {
  value?: string | string[];
  onChange: (value: string | string[]) => void;
  multiple?: boolean;
  maxFiles?: number;
  label?: string;
  hint?: string;
  error?: string;
  accept?: string;
  maxSizeMB?: number;
  className?: string;
}

// Determine asset type from MIME type
function getAssetType(mimeType: string): AssetType {
  if (mimeType.startsWith('video/')) return 'VIDEO';
  if (mimeType.startsWith('image/')) return 'IMAGE';
  return 'DOCUMENT';
}

// Compress image before upload (reduces sizes by 50-80%)
async function compressImage(file: File): Promise<File> {
  try {
    return await imageCompression(file, {
      maxSizeMB: 2,
      maxWidthOrHeight: 2000,
      useWebWorker: true,
      initialQuality: 0.8,
    });
  } catch {
    // If compression fails, use original file
    return file;
  }
}

// Check if a value is a permanent URL (not a tmp key)
function isPermanentUrl(value: string): boolean {
  return value.startsWith('http');
}

// Upload a file via presigned S3 URL. Returns { tmpKey, blobUrl }.
// The file stays in tmp/ until the entity is saved on the backend.
async function uploadFile(file: File): Promise<{ tmpKey: string; blobUrl: string }> {
  const assetType = getAssetType(file.type);

  // Compress images before upload (videos pass through as-is)
  let fileToUpload = file;
  if (assetType === 'IMAGE') {
    fileToUpload = await compressImage(file);
  }

  // Step 1: Get presigned upload URL for tmp/
  const { upload_url, tmp_key } = await assetsApi.getUploadUrl(
    file.name,
    assetType,
    fileToUpload.type,
    fileToUpload.size
  );

  // Step 2: PUT file directly to S3 tmp/ using presigned URL
  const putResponse = await fetch(upload_url, {
    method: 'PUT',
    body: fileToUpload,
    headers: {
      'Content-Type': fileToUpload.type,
    },
  });

  if (!putResponse.ok) {
    throw new Error(`S3 upload failed: ${putResponse.status} ${putResponse.statusText}`);
  }

  // Create a blob URL for local preview (no finalize call — backend does that on save)
  const blobUrl = URL.createObjectURL(fileToUpload);

  return { tmpKey: tmp_key, blobUrl };
}

export function ImageUpload({
  value,
  onChange,
  multiple = false,
  maxFiles = 5,
  label,
  hint,
  error,
  accept = 'image/*',
  maxSizeMB = 5,
  className,
}: ImageUploadProps) {
  const [isUploading, setIsUploading] = useState(false);
  const [dragActive, setDragActive] = useState(false);
  const [blobUrls, setBlobUrls] = useState<Map<string, string>>(new Map());
  const fileInputRef = useRef<HTMLInputElement>(null);

  const supportsVideo = accept.includes('video');

  // Clean up blob URLs on unmount
  useEffect(() => {
    return () => {
      blobUrls.forEach((blobUrl) => URL.revokeObjectURL(blobUrl));
    };
    // Only run cleanup on unmount
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Normalize value to array for easier handling - memoized to prevent useCallback dep changes
  const images = useMemo<string[]>(
    () => (Array.isArray(value) ? value : value ? [value] : []),
    [value]
  );

  // Resolve a value to a displayable src: permanent URLs pass through, tmp keys use blob URL
  const getDisplaySrc = useCallback(
    (val: string): string | undefined => {
      if (isPermanentUrl(val)) return val;
      return blobUrls.get(val);
    },
    [blobUrls]
  );

  const handleFiles = useCallback(
    async (files: FileList | null) => {
      if (!files || files.length === 0) return;

      const fileArray = Array.from(files);

      // Check max files
      if (multiple && images.length + fileArray.length > maxFiles) {
        toast.error(`Maximum ${maxFiles} files allowed`);
        return;
      }

      // Validate files
      for (const file of fileArray) {
        const isImage = file.type.startsWith('image/');
        const isVideo = file.type.startsWith('video/');

        if (!isImage && !isVideo) {
          toast.error(`${file.name} is not a supported file type`);
          return;
        }
        if (isVideo && !supportsVideo) {
          toast.error(`${file.name}: video uploads are not allowed here`);
          return;
        }
        if (file.size > maxSizeMB * 1024 * 1024) {
          toast.error(`${file.name} exceeds ${maxSizeMB}MB limit`);
          return;
        }
      }

      setIsUploading(true);

      try {
        const uploadedKeys: string[] = [];
        const newBlobEntries: [string, string][] = [];

        for (const file of fileArray) {
          const { tmpKey, blobUrl } = await uploadFile(file);
          uploadedKeys.push(tmpKey);
          newBlobEntries.push([tmpKey, blobUrl]);
        }

        // Store blob URLs for preview
        setBlobUrls((prev) => {
          const next = new Map(prev);
          for (const [key, url] of newBlobEntries) {
            next.set(key, url);
          }
          return next;
        });

        if (multiple) {
          onChange([...images, ...uploadedKeys]);
        } else {
          onChange(uploadedKeys[0]);
        }

        toast.success(`${fileArray.length > 1 ? 'Files' : 'File'} uploaded successfully`);
      } catch (err) {
        console.error('Upload error:', err);
        toast.error('Failed to upload file');
      } finally {
        setIsUploading(false);
      }
    },
    [images, maxFiles, maxSizeMB, multiple, onChange, supportsVideo]
  );

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setDragActive(false);
      handleFiles(e.dataTransfer.files);
    },
    [handleFiles]
  );

  const handleClick = () => {
    fileInputRef.current?.click();
  };

  const handleRemove = (index: number) => {
    const val = images[index];

    // Update form state immediately
    if (multiple) {
      const newImages = images.filter((_, i) => i !== index);
      onChange(newImages);
    } else {
      onChange('');
    }

    // Best-effort delete from S3 — only for permanent URLs (already in assets/)
    if (val && isPermanentUrl(val) && val.includes('.s3.amazonaws.com/assets/')) {
      assetsApi.delete(val).catch(() => {
        // Silently ignore — file stays in S3 but costs are minimal
      });
    }
    // For tmp keys: do nothing — S3 lifecycle auto-cleans in 24h

    // Revoke blob URL if it exists
    const blob = blobUrls.get(val);
    if (blob) {
      URL.revokeObjectURL(blob);
      setBlobUrls((prev) => {
        const next = new Map(prev);
        next.delete(val);
        return next;
      });
    }
  };

  const canAddMore = multiple ? images.length < maxFiles : images.length === 0;

  // Determine if a URL is a video (basic heuristic)
  const isVideoUrl = (url: string) => {
    const videoExtensions = ['.mp4', '.webm', '.ogg', '.mov', '.avi'];
    const lower = url.toLowerCase().split('?')[0];
    return videoExtensions.some((ext) => lower.endsWith(ext)) || lower.includes('/video/');
  };

  return (
    <div className={clsx('w-full', className)}>
      {label && <label className="label">{label}</label>}

      {/* Preview existing media */}
      {images.length > 0 && (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3 mb-3">
          {images.map((val, index) => {
            const src = getDisplaySrc(val);
            return (
              <div
                key={index}
                className="relative aspect-square rounded-lg overflow-hidden border border-gray-200 bg-gray-50"
              >
                {src ? (
                  isVideoUrl(val) ? (
                    <div className="w-full h-full flex items-center justify-center bg-gray-900">
                      <video src={src} className="w-full h-full object-cover" muted />
                      <Film className="absolute w-8 h-8 text-white opacity-75" />
                    </div>
                  ) : (
                    <img
                      src={src}
                      alt={`Preview ${index + 1}`}
                      className="w-full h-full object-cover"
                    />
                  )
                ) : (
                  <div className="w-full h-full flex items-center justify-center">
                    <Loader2 className="w-6 h-6 text-gray-400 animate-spin" />
                  </div>
                )}
                <button
                  type="button"
                  onClick={() => handleRemove(index)}
                  className="absolute top-1 right-1 p-1 bg-red-500 text-white rounded-full hover:bg-red-600 transition-colors"
                >
                  <X size={14} />
                </button>
              </div>
            );
          })}
        </div>
      )}

      {/* Upload area */}
      {canAddMore && (
        <div
          className={clsx(
            'relative border-2 border-dashed rounded-lg p-6 transition-colors cursor-pointer',
            dragActive
              ? 'border-primary-500 bg-primary-50'
              : 'border-gray-300 hover:border-gray-400',
            isUploading && 'pointer-events-none opacity-50'
          )}
          onDragEnter={handleDrag}
          onDragLeave={handleDrag}
          onDragOver={handleDrag}
          onDrop={handleDrop}
          onClick={handleClick}
        >
          <input
            ref={fileInputRef}
            type="file"
            accept={accept}
            multiple={multiple}
            onChange={(e) => handleFiles(e.target.files)}
            className="hidden"
          />

          <div className="flex flex-col items-center justify-center text-center">
            {isUploading ? (
              <>
                <Loader2 className="w-10 h-10 text-primary-500 animate-spin mb-2" />
                <p className="text-sm text-gray-600">Uploading...</p>
              </>
            ) : (
              <>
                <div className="w-12 h-12 rounded-full bg-gray-100 flex items-center justify-center mb-3">
                  {dragActive ? (
                    <ImageIcon className="w-6 h-6 text-primary-500" />
                  ) : (
                    <Upload className="w-6 h-6 text-gray-400" />
                  )}
                </div>
                <p className="text-sm font-medium text-gray-700">
                  {dragActive ? 'Drop file here' : 'Click to upload or drag and drop'}
                </p>
                <p className="text-xs text-gray-500 mt-1">
                  {supportsVideo ? 'Images or videos' : 'PNG, JPG, GIF'} up to {maxSizeMB}MB
                  {multiple && ` (max ${maxFiles} files)`}
                </p>
              </>
            )}
          </div>
        </div>
      )}

      {error && <p className="mt-1 text-sm text-red-600">{error}</p>}
      {hint && !error && <p className="mt-1 text-sm text-gray-500">{hint}</p>}
    </div>
  );
}
