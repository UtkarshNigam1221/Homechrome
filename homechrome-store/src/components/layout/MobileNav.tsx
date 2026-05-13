'use client';

import Image from 'next/image';
import Link from 'next/link';

import logo28 from '@/assets/logo-28.png';

import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { useAuthStore } from '@/stores/auth';
import { Category } from '@/types';

interface MobileNavProps {
  isOpen: boolean;
  onClose: () => void;
  categories: Category[];
}

export default function MobileNav({ isOpen, onClose, categories }: MobileNavProps) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const customer = useAuthStore((s) => s.customer);
  const logout = useAuthStore((s) => s.logout);

  return (
    <Sheet
      open={isOpen}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <SheetContent side="left" className="w-full max-w-xs p-0">
        <SheetHeader className="border-b border-border px-4 py-4">
          <SheetTitle>
            <Link href="/" className="flex items-center gap-2" onClick={onClose}>
              <Image src={logo28} alt="Homechrome" className="h-7 w-auto" unoptimized />
              <span className="text-lg font-bold tracking-tight text-foreground">
                HOME<span className="text-primary">CHROME</span>
              </span>
            </Link>
          </SheetTitle>
        </SheetHeader>

        {/* Navigation links */}
        <ScrollArea className="flex-1">
        <nav className="px-4 py-4">
          <div className="space-y-1">
            <SheetClose
              render={
                <Link
                  href="/products"
                  className="block rounded-lg px-3 py-2.5 text-base font-medium text-foreground transition-colors hover:bg-background"
                />
              }
            >
              All Products
            </SheetClose>
          </div>

          <div className="mt-6">
            <h3 className="px-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Categories
            </h3>
            <div className="mt-2 space-y-1">
              {categories.map((category) => (
                <SheetClose
                  key={category.id}
                  render={
                    <Link
                      href={`/c/${category.slug}`}
                      className="block rounded-lg px-3 py-2.5 text-base text-foreground transition-colors hover:bg-background"
                    />
                  }
                >
                  {category.name}
                </SheetClose>
              ))}
            </div>
          </div>
        </nav>
        </ScrollArea>

        {/* Account section */}
        <div className="border-t border-border px-4 py-4">
          {isAuthenticated ? (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Hello, {customer?.first_name || 'there'}
              </p>
              <SheetClose
                render={
                  <Link
                    href="/account"
                    className="block rounded-lg px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-background"
                  />
                }
              >
                My Account
              </SheetClose>
              <SheetClose
                render={
                  <Link
                    href="/account/orders"
                    className="block rounded-lg px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-background"
                  />
                }
              >
                My Orders
              </SheetClose>
              <Button
                variant="ghost"
                className="w-full justify-start text-red-600 hover:bg-red-50 hover:text-red-700"
                onClick={() => {
                  logout();
                  onClose();
                }}
              >
                Sign Out
              </Button>
            </div>
          ) : (
            <SheetClose
              render={
                <Link
                  href="/login"
                  className="block rounded-lg bg-primary px-4 py-2.5 text-center text-sm font-medium text-white transition-colors hover:bg-primary-dark"
                />
              }
            >
              Sign In
            </SheetClose>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
