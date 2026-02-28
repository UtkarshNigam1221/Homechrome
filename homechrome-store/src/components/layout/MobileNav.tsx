'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';

import { api } from '@/lib/api';
import { useAuthStore } from '@/stores/auth';
import { Category } from '@/types';

interface MobileNavProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function MobileNav({ isOpen, onClose }: MobileNavProps) {
  const { isAuthenticated, customer, logout } = useAuthStore();
  const [categories, setCategories] = useState<Category[]>([]);

  useEffect(() => {
    let cancelled = false;
    api.get<Category[]>('/api/v1/store/catalog/categories').then((res) => {
      if (!cancelled) setCategories(res.data);
    }).catch(() => {});
    return () => { cancelled = true; };
  }, []);

  // Prevent body scroll when menu is open
  useEffect(() => {
    if (isOpen) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }
    return () => {
      document.body.style.overflow = '';
    };
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 lg:hidden">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/40"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Slide-out panel */}
      <div className="fixed inset-y-0 left-0 w-full max-w-xs bg-white shadow-xl">
        <div className="flex h-full flex-col">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-border px-4 py-4">
            <span className="text-lg font-bold tracking-tight text-foreground">
              HOME<span className="text-primary">CHROME</span>
            </span>
            <button
              type="button"
              onClick={onClose}
              className="text-foreground"
              aria-label="Close menu"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
                strokeWidth={1.5}
                stroke="currentColor"
                className="h-6 w-6"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M6 18 18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>

          {/* Navigation links */}
          <nav className="flex-1 overflow-y-auto px-4 py-4">
            <div className="space-y-1">
              <Link
                href="/products"
                onClick={onClose}
                className="block rounded-lg px-3 py-2.5 text-base font-medium text-foreground transition-colors hover:bg-background"
              >
                All Products
              </Link>
            </div>

            <div className="mt-6">
              <h3 className="px-3 text-xs font-semibold uppercase tracking-wider text-muted">
                Categories
              </h3>
              <div className="mt-2 space-y-1">
                {categories.map((category) => (
                  <Link
                    key={category.id}
                    href={`/c/${category.slug}`}
                    onClick={onClose}
                    className="block rounded-lg px-3 py-2.5 text-base text-foreground transition-colors hover:bg-background"
                  >
                    {category.name}
                  </Link>
                ))}
              </div>
            </div>
          </nav>

          {/* Account section */}
          <div className="border-t border-border px-4 py-4">
            {isAuthenticated ? (
              <div className="space-y-3">
                <p className="text-sm text-muted">
                  Hello, {customer?.first_name || 'there'}
                </p>
                <Link
                  href="/account"
                  onClick={onClose}
                  className="block rounded-lg px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-background"
                >
                  My Account
                </Link>
                <Link
                  href="/account/orders"
                  onClick={onClose}
                  className="block rounded-lg px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-background"
                >
                  My Orders
                </Link>
                <button
                  type="button"
                  onClick={() => {
                    logout();
                    onClose();
                  }}
                  className="block w-full rounded-lg px-3 py-2 text-left text-sm font-medium text-red-600 transition-colors hover:bg-red-50"
                >
                  Sign Out
                </button>
              </div>
            ) : (
              <Link
                href="/login"
                onClick={onClose}
                className="block rounded-lg bg-primary px-4 py-2.5 text-center text-sm font-medium text-white transition-colors hover:bg-primary-dark"
              >
                Sign In
              </Link>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
