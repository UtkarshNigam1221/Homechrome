'use client';

import { MapPinIcon, ShoppingBagIcon, TruckIcon } from '@heroicons/react/24/outline';
import Link from 'next/link';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatPrice } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';

const navCards = [
  {
    href: '/account/orders',
    icon: ShoppingBagIcon,
    title: 'My Orders',
    description: 'View your order history and track shipments',
  },
  {
    href: '/account/addresses',
    icon: MapPinIcon,
    title: 'Addresses',
    description: 'Manage your saved delivery addresses',
  },
  {
    href: '/track',
    icon: TruckIcon,
    title: 'Track Order',
    description: 'Track your order with an order number',
  },
];

export default function AccountPage() {
  const customer = useAuthStore((s) => s.customer);

  if (!customer) return null;

  const addressCount = customer.addresses?.length ?? 0;

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Profile Information</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <p className="text-sm text-muted-foreground">Name</p>
              <p className="font-medium text-foreground">
                {customer.first_name} {customer.last_name}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Email</p>
              <p className="font-medium text-foreground">
                {customer.email || 'Not set'}
              </p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Phone</p>
              <p className="font-medium text-foreground">{customer.phone}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Saved Addresses</p>
              <p className="font-medium text-foreground">
                {addressCount} {addressCount === 1 ? 'address' : 'addresses'}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {navCards.map((card) => (
          <Link
            key={card.href}
            href={card.href}
            className="rounded-xl border border-foreground/10 bg-card p-5 transition-colors hover:border-primary/50"
          >
            <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <card.icon className="h-5 w-5 text-primary" />
            </div>
            <h3 className="font-semibold text-foreground">{card.title}</h3>
            <p className="mt-1 text-sm text-muted-foreground">{card.description}</p>
          </Link>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Account Summary</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4">
            <div className="rounded-lg bg-background p-4 text-center">
              <p className="text-2xl font-bold text-primary">{customer.total_orders}</p>
              <p className="text-sm text-muted-foreground">Total Orders</p>
            </div>
            <div className="rounded-lg bg-background p-4 text-center">
              <p className="text-2xl font-bold text-primary">
                {formatPrice(customer.total_spent)}
              </p>
              <p className="text-sm text-muted-foreground">Total Spent</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
