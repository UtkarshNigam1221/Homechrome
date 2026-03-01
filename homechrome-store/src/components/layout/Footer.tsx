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

          {/* Customer */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wider text-foreground">
              Customer
            </h3>
            <ul className="mt-3 space-y-2">
              <li>
                <Link
                  href="/track"
                  className="text-sm text-muted transition-colors hover:text-primary"
                >
                  Track Order
                </Link>
              </li>
              <li>
                <Link
                  href="/account"
                  className="text-sm text-muted transition-colors hover:text-primary"
                >
                  My Account
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
