'use client';

import { MagnifyingGlassPlusIcon, PhotoIcon } from '@heroicons/react/24/outline';
import { PlayIcon } from '@heroicons/react/24/solid';
import { AspectRatio, Box, Card, Center, Group, Image as MantineImage, Modal, ScrollArea, ThemeIcon, UnstyledButton } from '@mantine/core';
import { Carousel } from '@mantine/carousel';
import { useDisclosure } from '@mantine/hooks';
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
  const [opened, { open, close }] = useDisclosure(false);

  useEffect(() => {
    if (embla && embla.selectedScrollSnap() !== selectedIndex) {
      embla.scrollTo(selectedIndex);
    }
  }, [embla, selectedIndex]);

  return (
    <Box>
      <Card radius="lg" padding={0} bg="gray.1" withBorder={false} shadow="sm" style={{ overflow: 'hidden' }}>
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
                <GalleryMedia item={item} productName={productName} onOpen={open} />
              </Carousel.Slide>
            ))}
          </Carousel>
        ) : (
          <GalleryMedia item={selectedItem} productName={productName} onOpen={open} />
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

      <Modal
        opened={opened}
        onClose={close}
        fullScreen
        padding={0}
        withCloseButton
        title={null}
        styles={{
          content: { background: '#0A0E22' },
          header: { background: 'transparent', position: 'absolute', right: 0, top: 0, zIndex: 2 },
          close: { color: 'white', backgroundColor: 'rgba(255,255,255,0.12)' },
          body: { height: '100%', padding: 0 },
        }}
      >
        {hasMultiple ? (
          <Carousel
            initialSlide={selectedIndex}
            onSlideChange={onSelect}
            slideSize="100%"
            slideGap={0}
            emblaOptions={{ loop: true }}
            controlSize={44}
            height="100dvh"
            styles={{ viewport: { height: '100dvh' }, container: { height: '100dvh' } }}
          >
            {items.map((item, index) => (
              <Carousel.Slide key={item.kind === 'video' ? 'video' : `full-${index}`}>
                <Center h="100dvh" p="md">
                  <GalleryMedia item={item} productName={productName} fullRes />
                </Center>
              </Carousel.Slide>
            ))}
          </Carousel>
        ) : (
          <Center h="100dvh" p="md">
            <GalleryMedia item={selectedItem} productName={productName} fullRes />
          </Center>
        )}
      </Modal>
    </Box>
  );
}

interface GalleryMediaProps {
  item: GalleryItem | null;
  productName: string;
  onOpen?: () => void;
  fullRes?: boolean;
}

function GalleryMedia({ item, productName, onOpen, fullRes }: GalleryMediaProps) {
  const size = fullRes ? 1920 : 1080;
  const sizes = fullRes ? '100vw' : '(max-width: 1024px) 100vw, 50vw';
  const fit = fullRes ? 'contain' : 'cover';

  const content =
    item?.kind === 'video' ? (
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
        sizes={sizes}
        width={size}
        height={size}
        loading="eager"
        style={{ width: '100%', height: '100%', objectFit: fit }}
      />
    ) : (
      <Center bg="brand.1" h="100%">
        <ThemeIcon size={64} radius="xl" variant="light" color="brand">
          <PhotoIcon width={36} height={36} />
        </ThemeIcon>
      </Center>
    );

  if (fullRes) return content;

  // Inline square viewer; clicking an image opens the fullscreen lightbox.
  if (onOpen && item?.kind === 'image') {
    return (
      <AspectRatio ratio={1}>
        <UnstyledButton
          onClick={onOpen}
          aria-label="View image full screen"
          style={{ cursor: 'zoom-in' }}
        >
          {content}
          <ThemeIcon
            variant="filled"
            color="navy"
            radius="xl"
            size="lg"
            style={{ position: 'absolute', right: 12, bottom: 12, opacity: 0.85 }}
          >
            <MagnifyingGlassPlusIcon width={18} height={18} />
          </ThemeIcon>
        </UnstyledButton>
      </AspectRatio>
    );
  }

  return <AspectRatio ratio={1}>{content}</AspectRatio>;
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
