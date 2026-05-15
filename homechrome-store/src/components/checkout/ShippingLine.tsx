import { formatPrice } from '@/lib/utils';

interface Props {
  chargePaise: number | undefined;
  pendingLabel?: string;
}

export default function ShippingLine({
  chargePaise,
  pendingLabel = 'Calculated at checkout',
}: Props) {
  let display: React.ReactNode;
  if (chargePaise === undefined) {
    display = <span className="text-muted-foreground">{pendingLabel}</span>;
  } else if (chargePaise === 0) {
    display = <span className="text-green-600">Free</span>;
  } else {
    display = <span className="text-foreground">{formatPrice(chargePaise)}</span>;
  }
  return (
    <div className="flex justify-between text-sm">
      <span className="text-muted-foreground">Shipping</span>
      {display}
    </div>
  );
}
