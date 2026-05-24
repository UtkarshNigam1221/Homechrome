'use client';

import { PhotoIcon } from '@heroicons/react/24/outline';
import { PlayIcon } from '@heroicons/react/24/solid';
import { AspectRatio, Box, Center, Group, ScrollArea, UnstyledButton } from '@mantine/core';
import Image from 'next/image';

import { GalleryItem } from '@/hooks/useProductGallery';

interface ProductGalleryProps {
  productName: string;
  items: GalleryItem[];
  selectedIndex: number;
  selectedItem: GalleryItem | null;
  onSelect: (index: number) => void;
}

export function ProductGallery({
  productName,
  items,
  selectedIndex,
  selectedItem,
  onSelect,
}: ProductGalleryProps) {
  return (
    <Box>
      <Box style={{ overflow: 'hidden', borderRadius: 'var(--mantine-radius-lg)' }} bg="gray.1">
        <AspectRatio ratio={1}>
          {selectedItem?.kind === 'video' ? (
            <video
              key={selectedItem.url}
              src={selectedItem.url}
              controls
              playsInline
              preload="metadata"
              poster={selectedItem.poster}
              style={{ width: '100%', height: '100%', objectFit: 'contain' }}
            />
          ) : selectedItem?.kind === 'image' ? (
            <Image
              src={selectedItem.image.url}
              alt={selectedItem.image.alt_text || productName}
              fill
              sizes="(max-width: 1024px) 100vw, 50vw"
              style={{ objectFit: 'cover' }}
              priority
            />
          ) : (
            <Center bg="brand.1" h="100%">
              <PhotoIcon width={64} height={64} color="var(--mantine-color-brand-5)" opacity={0.4} />
            </Center>
          )}
        </AspectRatio>
      </Box>

      {items.length > 1 && (
        <ScrollArea scrollbarSize={6} mt="md" type="hover">
          <Group gap="sm" wrap="nowrap" pb="xs">
            {items.map((item, index) => (
              <GalleryThumbnail
                key={item.kind === 'video' ? 'video' : `img-${index}`}
                item={item}
                index={index}
                productName={productName}
                isActive={selectedIndex === index}
                onSelect={onSelect}
              />
            ))}
          </Group>
        </ScrollArea>
      )}
    </Box>
  );
}

interface ThumbProps {
  item: GalleryItem;
  index: number;
  productName: string;
  isActive: boolean;
  onSelect: (index: number) => void;
}

function GalleryThumbnail({ item, index, productName, isActive, onSelect }: ThumbProps) {
  return (
    <UnstyledButton
      onClick={() => onSelect(index)}
      aria-label={item.kind === 'video' ? 'Play product video' : `Image ${index + 1}`}
      style={{
        position: 'relative',
        width: 80,
        height: 80,
        flexShrink: 0,
        overflow: 'hidden',
        borderRadius: 'var(--mantine-radius-md)',
        border: `2px solid ${isActive ? 'var(--mantine-color-brand-5)' : 'transparent'}`,
        transition: 'border-color 0.15s',
      }}
    >
      {item.kind === 'video' ? (
        <>
          {item.poster ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={item.poster}
              alt="Product video thumbnail"
              style={{ width: '100%', height: '100%', objectFit: 'cover' }}
            />
          ) : (
            <Box w="100%" h="100%" bg="gray.2" />
          )}
          <Center pos="absolute" inset={0} bg="rgba(28,41,81,0.3)">
            <PlayIcon width={24} height={24} color="white" />
          </Center>
        </>
      ) : (
        <Image
          src={item.image.url}
          alt={item.image.alt_text || `${productName} ${index + 1}`}
          fill
          sizes="80px"
          style={{ objectFit: 'cover' }}
        />
      )}
    </UnstyledButton>
  );
}
