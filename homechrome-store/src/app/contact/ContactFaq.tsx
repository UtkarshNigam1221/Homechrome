'use client';

import { ChevronDownIcon } from '@heroicons/react/24/outline';
import { Accordion, Text } from '@mantine/core';

import { displayFont } from '../fonts';

/**
 * Client-side because Accordion's compound parts (Accordion.Item and friends)
 * come back undefined when read off a client component from a server one.
 */
export function ContactFaq({ faqs }: { faqs: { q: string; a: string }[] }) {
  return (
    <Accordion
      variant="separated"
      radius="lg"
      chevron={<ChevronDownIcon width={18} height={18} strokeWidth={2} />}
    >
      {faqs.map(({ q, a }) => (
        <Accordion.Item key={q} value={q}>
          <Accordion.Control>
            <Text
              fz={{ base: 'md', sm: '1.125rem' }}
              fw={600}
              c="navy.9"
              style={{ fontFamily: displayFont.style.fontFamily }}
            >
              {q}
            </Text>
          </Accordion.Control>
          <Accordion.Panel>
            <Text fz="sm" c="navy.7" lh={1.7}>
              {a}
            </Text>
          </Accordion.Panel>
        </Accordion.Item>
      ))}
    </Accordion>
  );
}
