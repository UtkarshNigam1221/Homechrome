import OrderCardSkeleton from '@/components/skeleton/OrderCardSkeleton';

export default function OrdersLoading() {
  return (
    <div className="space-y-4">
      <OrderCardSkeleton count={3} />
    </div>
  );
}
