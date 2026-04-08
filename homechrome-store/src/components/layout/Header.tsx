'use client';

import { Bars3Icon, ShoppingBagIcon, UserIcon } from '@heroicons/react/24/outline';
import dynamic from 'next/dynamic';
import Image from 'next/image';
import Link from 'next/link';

import logo32 from '@/assets/logo-32.png';
import logo40 from '@/assets/logo-40.png';
import { useRouter } from 'next/navigation';
import { useCallback, useState } from 'react';

import { Button } from '@/components/ui/button';
import { SearchInput } from '@/components/ui/search-input';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { useAuthStore } from '@/stores/auth';
import { useCartStore } from '@/stores/cart';

const MobileNav = dynamic(() => import('./MobileNav'), { ssr: false });

export default function Header() {
  const router = useRouter();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const customer = useAuthStore((s) => s.customer);
  const cartCount = useCartStore((s) => s.itemCount);
  const [searchQuery, setSearchQuery] = useState('');
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleSearch = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      if (searchQuery.trim()) {
        router.push(`/products?search=${encodeURIComponent(searchQuery.trim())}`);
        setSearchQuery('');
      }
    },
    [searchQuery, router],
  );

  return (
    <>
      <header className="sticky top-0 z-40 border-b border-border bg-white/95 backdrop-blur-sm">
        {/* Main header */}
        <div className="mx-auto flex max-w-7xl items-center justify-between gap-3 px-4 py-2 sm:gap-4 sm:px-6 sm:py-3 lg:px-8">
          {/* Left: hamburger + logo */}
          <div className="flex items-center gap-2 sm:gap-3">
            <Button
              variant="ghost"
              size="icon-sm"
              className="lg:hidden"
              onClick={() => setMobileMenuOpen(true)}
              aria-label="Open menu"
            >
              <Bars3Icon className="h-5 w-5" />
            </Button>

            <Link href="/" className="flex flex-shrink-0 items-center gap-2">
              <Image
                src={logo32}
                alt="Homechrome"
                className="h-8 w-auto sm:hidden"
                priority
                unoptimized
              />
              <Image
                src={logo40}
                alt="Homechrome"
                className="hidden h-10 w-auto sm:block"
                priority
                unoptimized
              />
              <span className="hidden text-lg font-bold tracking-tight text-foreground sm:inline sm:text-xl">
                HOME<span className="text-primary">CHROME</span>
              </span>
            </Link>
          </div>

          {/* Center: search */}
          <SearchInput
            value={searchQuery}
            onChange={setSearchQuery}
            onSubmit={handleSearch}
            placeholder="Search handloom textiles..."
            className="hidden max-w-md flex-1 lg:block"
          />

          {/* Right: nav + actions */}
          <div className="flex items-center gap-3 sm:gap-4">
            <nav className="hidden items-center gap-5 lg:flex">
              <Link
                href="/products"
                className="text-sm font-medium text-foreground transition-colors hover:text-primary"
              >
                Shop
              </Link>
              <Link
                href="/categories"
                className="text-sm font-medium text-foreground transition-colors hover:text-primary"
              >
                Categories
              </Link>
            </nav>

            {isAuthenticated ? (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Link
                      href="/account"
                      className="hidden text-foreground transition-colors hover:text-primary sm:block"
                    />
                  }
                >
                  <UserIcon className="h-5 w-5 sm:h-6 sm:w-6" />
                </TooltipTrigger>
                <TooltipContent>{customer?.first_name || 'Account'}</TooltipContent>
              </Tooltip>
            ) : (
              <Link
                href="/login"
                className="hidden text-sm font-medium text-foreground transition-colors hover:text-primary sm:block"
              >
                Login
              </Link>
            )}

            <Tooltip>
              <TooltipTrigger
                render={<Link href="/cart" className="relative" />}
              >
                <ShoppingBagIcon className="h-5 w-5 text-foreground transition-colors hover:text-primary sm:h-6 sm:w-6" />
                {cartCount > 0 && (
                  <span className="absolute -right-1.5 -top-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-primary text-[10px] font-bold text-white sm:h-5 sm:w-5 sm:text-xs">
                    {cartCount}
                  </span>
                )}
              </TooltipTrigger>
              <TooltipContent>
                {cartCount > 0 ? `Cart (${cartCount})` : 'Cart'}
              </TooltipContent>
            </Tooltip>
          </div>
        </div>

        {/* Mobile search bar */}
        <div className="border-t border-border px-4 pb-2 pt-1 lg:hidden">
          <SearchInput
            value={searchQuery}
            onChange={setSearchQuery}
            onSubmit={handleSearch}
            placeholder="Search handloom textiles..."
          />
        </div>
      </header>

      <MobileNav
        isOpen={mobileMenuOpen}
        onClose={() => setMobileMenuOpen(false)}
      />
    </>
  );
}
