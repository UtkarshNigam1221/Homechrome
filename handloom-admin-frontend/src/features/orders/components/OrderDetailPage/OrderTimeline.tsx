import { format } from 'date-fns';
import { Clock } from 'lucide-react';

import { Card } from '@/shared/components/ui';

interface OrderTimelineProps {
  createdAt: string;
  updatedAt: string;
}

export function OrderTimeline({ createdAt, updatedAt }: OrderTimelineProps) {
  return (
    <Card>
      <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
        <Clock className="w-5 h-5" />
        Timeline
      </h2>
      <div className="space-y-3">
        <div className="flex items-start gap-3">
          <div className="w-2 h-2 bg-primary-600 rounded-full mt-2" />
          <div>
            <p className="text-sm font-medium">Order Placed</p>
            <p className="text-xs text-gray-500">
              {format(new Date(createdAt), 'MMM d, yyyy h:mm a')}
            </p>
          </div>
        </div>
        {updatedAt !== createdAt && (
          <div className="flex items-start gap-3">
            <div className="w-2 h-2 bg-gray-400 rounded-full mt-2" />
            <div>
              <p className="text-sm font-medium">Last Updated</p>
              <p className="text-xs text-gray-500">
                {format(new Date(updatedAt), 'MMM d, yyyy h:mm a')}
              </p>
            </div>
          </div>
        )}
      </div>
    </Card>
  );
}
