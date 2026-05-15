import { Badge } from '@/shared/components/ui';

import type { ShipmentPriority } from '../types';

interface PriorityBadgeProps {
  priority: ShipmentPriority;
}

export function PriorityBadge({ priority }: Readonly<PriorityBadgeProps>) {
  return (
    <Badge variant={priority === 'PRIORITY' ? 'warning' : 'gray'}>
      {priority === 'PRIORITY' ? 'Priority' : 'Normal'}
    </Badge>
  );
}
