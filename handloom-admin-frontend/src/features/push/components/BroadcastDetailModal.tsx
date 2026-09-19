import { format } from 'date-fns';

import { Badge, Modal } from '@/shared/components/ui';

import { broadcastStatusVariant } from '../lib/statusVariant';
import type { PushBroadcast } from '../types';

interface BroadcastDetailModalProps {
  broadcast: PushBroadcast | null;
  onClose: () => void;
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[7rem_1fr] gap-3 py-2">
      <dt className="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</dt>
      <dd className="min-w-0 break-words text-sm text-gray-900">{children}</dd>
    </div>
  );
}

/** Everything the fan-out recorded, including the fields the list row hides. */
export function BroadcastDetailModal({ broadcast, onClose }: BroadcastDetailModalProps) {
  return (
    <Modal isOpen={broadcast !== null} onClose={onClose} title="Broadcast" size="lg">
      {broadcast && (
        <dl className="divide-y divide-gray-100">
          <Row label="Status">
            <div className="flex items-center gap-2">
              <Badge variant={broadcastStatusVariant[broadcast.status]}>{broadcast.status}</Badge>
              <span className="text-gray-600">
                {broadcast.success_count} delivered, {broadcast.failure_count} failed, of{' '}
                {broadcast.total_targeted} targeted
              </span>
            </div>
          </Row>
          <Row label="Title">{broadcast.title}</Row>
          <Row label="Body">{broadcast.body}</Row>
          <Row label="Opens">{broadcast.url}</Row>
          {broadcast.tag && <Row label="Tag">{broadcast.tag}</Row>}
          {broadcast.image && (
            <Row label="Banner">
              <img
                src={broadcast.image}
                alt=""
                className="mb-1 max-h-32 rounded-md border border-gray-200 object-cover"
              />
              <span className="text-xs text-gray-500">{broadcast.image}</span>
            </Row>
          )}
          {broadcast.icon && (
            <Row label="Icon">
              <img
                src={broadcast.icon}
                alt=""
                className="mb-1 h-10 w-10 rounded-md border border-gray-200 object-cover"
              />
              <span className="text-xs text-gray-500">{broadcast.icon}</span>
            </Row>
          )}
          {broadcast.actions && broadcast.actions.length > 0 && (
            <Row label="Buttons">
              <ul className="space-y-1">
                {broadcast.actions.map((action) => (
                  <li key={action.action}>
                    {action.title}
                    {action.url && <span className="text-gray-500"> → {action.url}</span>}
                  </li>
                ))}
              </ul>
            </Row>
          )}
          <Row label="Sent">
            {format(new Date(broadcast.sent_at), 'dd MMM yyyy, HH:mm:ss')} by {broadcast.sent_by}
          </Row>
          <Row label="ID">
            <span className="font-mono text-xs text-gray-600">{broadcast.id}</span>
          </Row>
        </dl>
      )}
    </Modal>
  );
}
