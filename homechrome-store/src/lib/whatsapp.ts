import { SUPPORT_WHATSAPP } from './constants';

/** One place that knows how a wa.me link is shaped. */
export function whatsappHref(message: string): string {
  return `https://wa.me/${SUPPORT_WHATSAPP}?text=${encodeURIComponent(message)}`;
}
