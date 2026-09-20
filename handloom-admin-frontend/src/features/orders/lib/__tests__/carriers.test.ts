import { describe, expect, it } from 'vitest';

import { CARRIER_OPTIONS, CARRIER_TRACKING_URLS, carrierOptions } from '../carriers';

describe('carrierOptions', () => {
  it('offers the carriers we ship with', () => {
    expect(CARRIER_OPTIONS.map((o) => o.value)).toEqual(['DTDC']);
  });

  it('keeps a carrier recorded before it was in the list', () => {
    // Dropping it would rewrite the courier to DTDC on the next save.
    const options = carrierOptions('Blue Dart');
    expect(options.map((o) => o.value)).toEqual(['Blue Dart', 'DTDC']);
    expect(options[0].label).toMatch(/no longer offered/i);
  });

  it('does not duplicate a carrier that is still offered', () => {
    expect(carrierOptions('DTDC')).toEqual(CARRIER_OPTIONS);
  });

  it('tracks every offered carrier on an https page', () => {
    for (const url of Object.values(CARRIER_TRACKING_URLS)) {
      expect(url).toMatch(/^https:\/\/\S+$/);
    }
  });
});
