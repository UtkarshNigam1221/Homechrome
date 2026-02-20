import Link from 'next/link';

export default function Footer() {
  return (
    <footer className="border-t border-border bg-white">
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-4">
          {/* Brand */}
          <div>
            <span className="text-xl font-bold tracking-tight text-foreground">
              HOME<span className="text-primary">CHROME</span>
            </span>
            <p className="mt-3 text-sm leading-relaxed text-muted">
              Premium handloom textiles from master artisans across India.
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
                  className="text-sm text-muted transition-colors hover:text-primary"
                >
                  All Products
                </Link>
              </li>
              <li>
                <Link
                  href="/categories"
                  className="text-sm text-muted transition-colors hover:text-primary"
                >
                  Categories
                </Link>
              </li>
            </ul>
          </div>

          {/* Company */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-foreground">
              Company
            </h3>
            <ul className="mt-3 space-y-2">
              <li>
                <Link
                  href="/about"
                  className="text-sm text-muted transition-colors hover:text-primary"
                >
                  About Us
                </Link>
              </li>
              <li>
                <Link
                  href="/contact"
                  className="text-sm text-muted transition-colors hover:text-primary"
                >
                  Contact
                </Link>
              </li>
            </ul>
          </div>

          {/* Legal */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-foreground">
              Legal
            </h3>
            <ul className="mt-3 space-y-2">
              <li>
                <Link
                  href="/terms"
                  className="text-sm text-muted transition-colors hover:text-primary"
                >
                  Terms of Service
                </Link>
              </li>
              <li>
                <Link
                  href="/privacy"
                  className="text-sm text-muted transition-colors hover:text-primary"
                >
                  Privacy Policy
                </Link>
              </li>
              <li>
                <Link
                  href="/refund-policy"
                  className="text-sm text-muted transition-colors hover:text-primary"
                >
                  Refund Policy
                </Link>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-10 border-t border-border pt-6">
          <p className="text-center text-xs text-muted">
            &copy; {new Date().getFullYear()} Homechrome. All rights reserved.
          </p>
        </div>
      </div>
    </footer>
  );
}
