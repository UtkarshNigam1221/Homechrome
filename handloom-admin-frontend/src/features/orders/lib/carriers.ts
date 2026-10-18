/**
 * Couriers we ship with, and the page a customer tracks a shipment on. Add an
 * entry to offer a new carrier — the dropdown and the tracking link both read
 * from here.
 */
export const CARRIER_TRACKING_URLS: Record<string, string> = {
  DTDC: 'https://www.dtdc.com/track-your-shipment/',
};

export const CARRIER_OPTIONS = Object.keys(CARRIER_TRACKING_URLS).map((carrier) => ({
  value: carrier,
  label: carrier,
}));

/**
 * The options to show for an order, keeping a carrier recorded before it was in
 * the list. Dropping it would silently rewrite the courier on the next save.
 */
export function carrierOptions(current?: string) {
  if (!current || current in CARRIER_TRACKING_URLS) return CARRIER_OPTIONS;
  return [{ value: current, label: `${current} (no longer offered)` }, ...CARRIER_OPTIONS];
}
