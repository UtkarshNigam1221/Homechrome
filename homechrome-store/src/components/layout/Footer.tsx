import Image from 'next/image';
import Link from 'next/link';

import logo40 from '@/assets/logo-40.png';

import { Container } from '@/components/ui/container';
import { Separator } from '@/components/ui/separator';

export default function Footer() {
  return (
    <footer className="border-t border-border bg-white">
      <Container className="py-12">
        <div className="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-4">
          {/* Brand */}
          <div>
            <Link href="/" className="inline-flex items-center gap-2.5">
              <Image src={logo40} alt="Homechrome" className="h-10 w-auto" unoptimized />
              <span className="text-xl font-bold tracking-tight text-foreground">
                HOME<span className="text-primary">CHROME</span>
              </span>
            </Link>
            <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
              Premium handloom textiles from across India.
              Celebrating the art of traditional weaving.
            </p>
          </div>

          {/* Shop */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-foreground">
              Shop
            </h3>
            <ul className="mt-3 space-y-2">
              <li>
                <Link
                  href="/products"
                  className="text-sm text-muted-foreground transition-colors hover:text-primary"
                >
                  All Products
                </Link>
              </li>
              <li>
                <Link
                  href="/categories"
                  className="text-sm text-muted-foreground transition-colors hover:text-primary"
                >
                  Categories
                </Link>
              </li>
            </ul>
          </div>

          {/* Customer */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-foreground">
              Customer
            </h3>
            <ul className="mt-3 space-y-2">
              <li>
                <Link
                  href="/track"
                  className="text-sm text-muted-foreground transition-colors hover:text-primary"
                >
                  Track Order
                </Link>
              </li>
              <li>
                <Link
                  href="/account"
                  className="text-sm text-muted-foreground transition-colors hover:text-primary"
                >
                  My Account
                </Link>
              </li>
              <li>
                <Link
                  href="/return-policy"
                  className="text-sm text-muted-foreground transition-colors hover:text-primary"
                >
                  Return Policy
                </Link>
              </li>
            </ul>
          </div>
        </div>

        <Separator className="mt-10 mb-6" />
        <p className="text-center text-xs text-muted-foreground">
          &copy; {new Date().getFullYear()} Homechrome. All rights reserved.
        </p>
      </Container>
    </footer>
  );
}
