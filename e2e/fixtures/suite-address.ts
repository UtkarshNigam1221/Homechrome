/**
 * What marks an address as this suite's, shared by the helper that creates one
 * and the script that reaps it. One definition on purpose: a marker that drifts
 * between the two is a leak cleanup cheerfully reports as zero.
 */

/** Same prefix the suite's products and categories carry. */
export const E2E_PREFIX = 'E2E-';

export const SUITE_ADDRESS_LINE1 = `${E2E_PREFIX}1 Test Street`;

export interface SuiteAddress {
  id?: string;
  first_name?: string;
  address_line1?: string;
}

/**
 * Also matches the rows written before the prefix existed, which used first_name
 * alone — those are the pile #283 is about, so cleanup has to reap them too.
 */
export function isSuiteAddress(address: SuiteAddress): boolean {
  return address.address_line1?.startsWith(E2E_PREFIX) === true || address.first_name === 'E2E';
}
