import { format } from 'date-fns';
import { MessageSquare } from 'lucide-react';

import { Badge, Card } from '@/shared/components/ui';

import type { OrderNote } from '../../types';

interface OrderNotesProps {
  notes: OrderNote[];
}

export function OrderNotes({ notes }: OrderNotesProps) {
  if (notes.length === 0) return null;

  return (
    <Card>
      <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
        <MessageSquare className="w-5 h-5" />
        Order Notes
      </h2>
      <div className="space-y-3">
        {notes.map((note, index) => (
          <div
            key={note.id || index}
            className={`p-3 rounded-lg ${note.is_internal ? 'bg-yellow-50 border border-yellow-200' : 'bg-gray-50'}`}
          >
            <p className="text-sm">{note.note}</p>
            <div className="flex items-center gap-2 mt-2 text-xs text-gray-500">
              <span>{note.created_by}</span>
              <span>&bull;</span>
              <span>{format(new Date(note.created_at), 'MMM d, yyyy h:mm a')}</span>
              {note.is_internal && (
                <Badge variant="warning" size="sm">
                  Internal
                </Badge>
              )}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}
