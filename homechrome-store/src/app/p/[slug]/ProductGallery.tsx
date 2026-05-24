'use client';

import { PhotoIcon } from '@heroicons/react/24/outline';
import { PlayIcon } from '@heroicons/react/24/solid';
import { AspectRatio, Box, Card, Center, Group, Image as MantineImage, ScrollArea, ThemeIcon, UnstyledButton } from '@mantine/core';
import { Carousel } from '@mantine/carousel';
import type { EmblaCarouselType } from 'embla-carousel';
import { useEffect, useState } from 'react';
import { AssetImage } from '@/components/ui/asset-image';

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
  const hasMultiple = items.length > 1;
  const [embla, setEmbla] = useState<EmblaCarouselType | null>(null);

  useEffect(() => {
    if (embla && embla.selectedScrollSnap() !== selectedIndex) {
      embla.scrollTo(selectedIndex);
    }
  }, [embla, selectedIndex]);

  return (
    <Box>
      <Card radius="lg" padding={0} bg="gray.1" withBorder={false}>
        {hasMultiple ? (
          <Carousel
            withIndicators
            withControls
            slideSize="100%"
            slideGap={0}
            initialSlide={selectedIndex}
            onSlideChange={onSelect}
            getEmblaApi={setEmbla}
            emblaOptions={{ loop: true }}
            controlSize={36}
          >
            {items.map((item, index) => (
              <Carousel.Slide key={item.kind === 'video' ? 'video' : `img-${index}`}>
                <GalleryMedia item={item} productName={productName} />
              </Carousel.Slide>
            ))}
          </Carousel>
        ) : (
          <GalleryMedia item={selectedItem} productName={productName} />
        )}
      </Card>

      {hasMultiple && (
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

interface GalleryMediaProps {
  item: GalleryItem | null;
  productName: string;
}

function GalleryMedia({ item, productName }: GalleryMediaProps) {
  return (
    <AspectRatio ratio={1}>
      {item?.kind === 'video' ? (
        <video
          key={item.url}
          src={item.url}
          poster={item.poster}
          controls
          playsInline
          preload="metadata"
          style={{ width: '100%', height: '100%', objectFit: 'contain' }}
        />
      ) : item?.kind === 'image' ? (
        <AssetImage
          src={item.image.url}
          alt={item.image.alt_text || productName}
          sizes="(max-width: 1024px) 100vw, 50vw"
          width={1080}
          height={1080}
          loading="eager"
          style={{ width: '100%', height: '100%', objectFit: 'cover' }}
        />
      ) : (
        <Center bg="brand.1">
          <ThemeIcon size={64} radius="xl" variant="light" color="brand">
            <PhotoIcon width={36} height={36} />
          </ThemeIcon>
        </Center>
      )}
    </AspectRatio>
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
      data-active={isActive || undefined}
    >
      <Card
        radius="md"
        padding={0}
        withBorder
        bg="gray.1"
        style={{
          width: 80,
          height: 80,
          overflow: 'hidden',
          borderColor: isActive ? 'var(--mantine-color-brand-5)' : 'transparent',
          borderWidth: 2,
        }}
      >
        {item.kind === 'video' ? (
          <Box pos="relative" w="100%" h="100%">
            {item.poster ? (
              <MantineImage src={item.poster} alt="Product video thumbnail" w="100%" h="100%" fit="cover" />
            ) : (
              <Box w="100%" h="100%" bg="gray.2" />
            )}
            <Center pos="absolute" inset={0} bg="rgba(28,41,81,0.3)">
              <PlayIcon width={24} height={24} color="white" />
            </Center>
          </Box>
        ) : (
          <AssetImage
            src={item.image.url}
            alt={item.image.alt_text || `${productName} ${index + 1}`}
            sizes="80px"
            width={80}
            height={80}
            style={{ width: '100%', height: '100%', objectFit: 'cover' }}
          />
        )}
      </Card>
    </UnstyledButton>
  );
}
