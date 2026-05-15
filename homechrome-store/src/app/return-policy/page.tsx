import type { Metadata } from 'next';

import { Container } from '@/components/ui/container';
import { PageHeader } from '@/components/ui/page-header';

// Fully static; no revalidate needed. If content moves to CMS, add `export const revalidate = ...`.

const SUPPORT_EMAIL = 'support@homechrome.in';

export const metadata: Metadata = {
  title: 'Return Policy | Homechrome',
  description:
    'Homechrome return policy — eligibility, return window, refund timelines, and how to raise a return for handloom products.',
};

export default function ReturnPolicyPage() {
  return (
    <Container size="narrow" className="max-w-3xl py-12">
      <PageHeader
        title="Return Policy"
        description="Everything you need to know about returns, refunds, and damaged items."
      />

      <div className="space-y-8 text-foreground">
        <section>
          <h2 className="text-lg font-semibold">Return window</h2>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            You can request a return within <strong>7 days of delivery</strong>.
            The product must be unused, unwashed, and in its original packaging
            with all tags intact.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">How to raise a return</h2>
          <ol className="mt-2 list-decimal space-y-2 pl-5 text-sm leading-relaxed text-muted-foreground">
            <li>
              Email us at{' '}
              <a
                href={`mailto:${SUPPORT_EMAIL}`}
                className="font-medium text-primary hover:text-primary-dark"
              >
                {SUPPORT_EMAIL}
              </a>{' '}
              with your order number and a brief reason for the return.
            </li>
            <li>
              Our team will confirm eligibility and schedule a free reverse
              pickup with our courier partner, Delhivery, within 2 business
              days.
            </li>
            <li>
              Keep the item packed in its original packaging. The courier
              executive will collect it from the delivery address.
            </li>
            <li>
              Once we receive and inspect the item at our warehouse, we will
              process your refund.
            </li>
          </ol>
        </section>

        <section>
          <h2 className="text-lg font-semibold">Refund timeline</h2>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            Refunds are issued to the original payment method within{' '}
            <strong>5–7 business days</strong> of the returned item being
            received and inspected. For cash-on-delivery orders, refunds are
            issued via bank transfer to an account you nominate.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">Damaged or defective items</h2>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            If the product arrives damaged, defective, or different from what
            you ordered, please email{' '}
            <a
              href={`mailto:${SUPPORT_EMAIL}`}
              className="font-medium text-primary hover:text-primary-dark"
            >
              {SUPPORT_EMAIL}
            </a>{' '}
            within <strong>48 hours of delivery</strong> with clear photos of
            the product and the packaging. We will arrange a free pickup and
            either replace the item or issue a full refund — your choice.
          </p>
        </section>

        <section>
          <h2 className="text-lg font-semibold">What cannot be returned</h2>
          <ul className="mt-2 list-disc space-y-2 pl-5 text-sm leading-relaxed text-muted-foreground">
            <li>Products marked as final sale or non-returnable at checkout.</li>
            <li>Items custom-sized or made-to-order.</li>
            <li>
              Items that show signs of use, washing, or damage caused after
              delivery.
            </li>
            <li>Items returned without their original packaging or tags.</li>
            <li>Items returned more than 7 days after delivery.</li>
          </ul>
        </section>

        <section>
          <h2 className="text-lg font-semibold">Need help?</h2>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
            We are happy to help with any questions about a return. Reach out
            to us at{' '}
            <a
              href={`mailto:${SUPPORT_EMAIL}`}
              className="font-medium text-primary hover:text-primary-dark"
            >
              {SUPPORT_EMAIL}
            </a>{' '}
            and we will get back within one business day.
          </p>
        </section>
      </div>
    </Container>
  );
}
