'use client';

import Link from 'next/link';

import { useAuthStore } from '@/stores/auth';

export default function AccountPage() {
  const customer = useAuthStore((s) => s.customer);

  if (!customer) return null;

  const addressCount = customer.addresses?.length ?? 0;

  return (
    <div className="space-y-6">
      <div className="rounded-lg border border-border bg-white p-6">
        <h2 className="mb-4 text-lg font-semibold text-foreground">
          Profile Information
        </h2>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <p className="text-sm text-muted">Name</p>
            <p className="font-medium text-foreground">
              {customer.first_name} {customer.last_name}
            </p>
          </div>
          <div>
            <p className="text-sm text-muted">Email</p>
            <p className="font-medium text-foreground">
              {customer.email || 'Not set'}
            </p>
          </div>
          <div>
            <p className="text-sm text-muted">Phone</p>
            <p className="font-medium text-foreground">{customer.phone}</p>
          </div>
          <div>
            <p className="text-sm text-muted">Saved Addresses</p>
            <p className="font-medium text-foreground">
              {addressCount} {addressCount === 1 ? 'address' : 'addresses'}
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <Link
          href="/account/orders"
          className="rounded-lg border border-border bg-white p-5 transition-colors hover:border-primary/50"
        >
          <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
              className="h-5 w-5 text-primary"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M15.75 10.5V6a3.75 3.75 0 1 0-7.5 0v4.5m11.356-1.993 1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 0 1-1.12-1.243l1.264-12A1.125 1.125 0 0 1 5.513 7.5h12.974c.576 0 1.059.435 1.119 1.007ZM8.625 10.5a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm7.5 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
              />
            </svg>
          </div>
          <h3 className="font-semibold text-foreground">My Orders</h3>
          <p className="mt-1 text-sm text-muted">
            View your order history and track shipments
          </p>
        </Link>

        <Link
          href="/account/addresses"
          className="rounded-lg border border-border bg-white p-5 transition-colors hover:border-primary/50"
        >
          <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
              className="h-5 w-5 text-primary"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M15 10.5a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"
              />
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1 1 15 0Z"
              />
            </svg>
          </div>
          <h3 className="font-semibold text-foreground">Addresses</h3>
          <p className="mt-1 text-sm text-muted">
            Manage your saved delivery addresses
          </p>
        </Link>

        <Link
          href="/track"
          className="rounded-lg border border-border bg-white p-5 transition-colors hover:border-primary/50"
        >
          <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
              className="h-5 w-5 text-primary"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M8.25 18.75a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m3 0h6m-9 0H3.375a1.125 1.125 0 0 1-1.125-1.125V14.25m17.25 4.5a1.5 1.5 0 0 1-3 0m3 0a1.5 1.5 0 0 0-3 0m3 0h1.125c.621 0 1.07-.504 1.004-1.12l-.894-8.378A1.125 1.125 0 0 0 18.38 8.25H16.5m-11.25 0V5.625c0-.621.504-1.125 1.125-1.125h5.25c.621 0 1.125.504 1.125 1.125v2.625m-7.5 0h7.5"
              />
            </svg>
          </div>
          <h3 className="font-semibold text-foreground">Track Order</h3>
          <p className="mt-1 text-sm text-muted">
            Track your order with an order number
          </p>
        </Link>
      </div>

      <div className="rounded-lg border border-border bg-white p-6">
        <h2 className="mb-2 text-lg font-semibold text-foreground">
          Account Summary
        </h2>
        <div className="grid grid-cols-2 gap-4">
          <div className="rounded-lg bg-background p-4 text-center">
            <p className="text-2xl font-bold text-primary">{customer.total_orders}</p>
            <p className="text-sm text-muted">Total Orders</p>
          </div>
          <div className="rounded-lg bg-background p-4 text-center">
            <p className="text-2xl font-bold text-primary">
              {`₹${(customer.total_spent / 100).toLocaleString('en-IN')}`}
            </p>
            <p className="text-sm text-muted">Total Spent</p>
          </div>
        </div>
      </div>
    </div>
  );
}
