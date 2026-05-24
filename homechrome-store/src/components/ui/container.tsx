'use client';

import { Container as MantineContainer, ContainerProps as MantineContainerProps } from '@mantine/core';

interface ContainerProps extends Omit<MantineContainerProps, 'size'> {
  size?: 'default' | 'narrow';
}

export function Container({ size = 'default', ...props }: ContainerProps) {
  return <MantineContainer size={size === 'default' ? 'xl' : 'lg'} {...props} />;
}
