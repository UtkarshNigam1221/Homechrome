import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Megaphone, Monitor, Send, Smartphone, Users } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { pushApi } from '@/features/push/api';
import { NotificationPreview } from '@/features/push/components/NotificationPreview';
import { getErrorMessage } from '@/shared/api/client';
import { PageLoading } from '@/shared/components/loading';
import { Badge, Button, Card, Input, PageHeader } from '@/shared/components/ui';

import type { PushBroadcastStatus, PushSubscriber } from '../types';

const TITLE_MAX = 120;
const BODY_MAX = 300;

const statusVariant: Record<PushBroadcastStatus, 'success' | 'warning' | 'danger'> = {
  SUCCESS: 'success',
  PARTIAL: 'warning',
  FAILED: 'danger',
};

function deviceLabel(subscriber: PushSubscriber): string {
  const { browser, os } = subscriber.device ?? {};
  if (!browser && !os) return 'Unknown device';
  return [browser, os].filter(Boolean).join(' · ');
}

export function PushBroadcastPage() {
  const queryClient = useQueryClient();

  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [url, setUrl] = useState('/products');
  const [tag, setTag] = useState('');
  const [image, setImage] = useState('');

  const { data: subscribersData, isLoading: subscribersLoading } = useQuery({
    queryKey: ['push-subscribers'],
    queryFn: () => pushApi.listSubscribers({ status: 'ACTIVE', limit: 50 }),
  });

  const { data: broadcastsData } = useQuery({
    queryKey: ['push-broadcasts'],
    queryFn: () => pushApi.listBroadcasts(20),
  });

  const broadcastMutation = useMutation({
    mutationFn: pushApi.broadcast,
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ['push-broadcasts'] });
      queryClient.invalidateQueries({ queryKey: ['push-subscribers'] });
      toast.success(
        `Sent to ${result.success_count} of ${result.total_targeted} device${
          result.total_targeted === 1 ? '' : 's'
        }`
      );
      setTitle('');
      setBody('');
      setTag('');
      setImage('');
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  if (subscribersLoading) {
    return <PageLoading />;
  }

  const subscribers = subscribersData?.subscriptions ?? [];
  const broadcasts = broadcastsData?.broadcasts ?? [];
  const canSend = title.trim().length > 0 && body.trim().length > 0;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSend) return;
    broadcastMutation.mutate({
      title: title.trim(),
      body: body.trim(),
      url: url.trim() || '/',
      tag: tag.trim() || undefined,
      image: image.trim() || undefined,
    });
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Push Broadcast"
        subtitle={`${subscribers.length} active subscriber${subscribers.length === 1 ? '' : 's'}`}
      />

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="flex items-center gap-2 text-gray-900">
              <Megaphone className="w-5 h-5 text-primary-600" />
              <h2 className="font-semibold">Compose</h2>
            </div>

            <Input
              label="Title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              maxLength={TITLE_MAX}
              placeholder="Festive Handloom Drop — 20% off"
              hint={`${title.length}/${TITLE_MAX}`}
              required
            />

            <div className="w-full">
              <label
                htmlFor="broadcast-body"
                className="block text-sm font-medium text-gray-700 mb-1"
              >
                Message
              </label>
              <textarea
                id="broadcast-body"
                value={body}
                onChange={(e) => setBody(e.target.value)}
                maxLength={BODY_MAX}
                rows={3}
                required
                placeholder="Rare Banarasi and Chanderi sarees, woven by master artisans."
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500"
              />
              <p className="mt-1 text-xs text-gray-500">
                {body.length}/{BODY_MAX}
              </p>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <Input
                label="Click-through URL"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="/products"
                hint="Storefront path opened when tapped"
              />
              <Input
                label="Replaces tag (optional)"
                value={tag}
                onChange={(e) => setTag(e.target.value)}
                placeholder="festive-drop"
                hint="Reusing a tag replaces that unread alert. Leave empty to stack."
              />
            </div>

            <Input
              label="Banner image (optional)"
              value={image}
              onChange={(e) => setImage(e.target.value)}
              placeholder="https://dev-store.homechrome.in/banners/festive.jpg"
              hint="Full URL to a wide image, shown when the notification is expanded. Around 2:1 works best."
            />

            <div className="flex items-center justify-between border-t border-gray-200 pt-4">
              <p className="text-sm text-gray-500">
                Sends to all {subscribers.length} active device
                {subscribers.length === 1 ? '' : 's'}. This cannot be undone.
              </p>
              <Button
                type="submit"
                leftIcon={<Send className="w-4 h-4" />}
                loading={broadcastMutation.isPending}
                disabled={!canSend || subscribers.length === 0}
              >
                Send broadcast
              </Button>
            </div>
          </form>
        </Card>

        <div className="space-y-6">
          <Card>
            <NotificationPreview title={title} body={body} image={image.trim() || undefined} />
          </Card>

          <Card>
            <div className="flex items-center gap-2 text-gray-900 mb-4">
              <Users className="w-5 h-5 text-primary-600" />
              <h2 className="font-semibold">Subscribers</h2>
            </div>

            {subscribers.length === 0 ? (
              <p className="text-sm text-gray-500">
                No active subscribers yet. Visitors opt in from the storefront.
              </p>
            ) : (
              <ul className="divide-y divide-gray-200">
                {subscribers.map((subscriber) => (
                  <li key={subscriber.id} className="flex items-center gap-3 py-3">
                    {subscriber.device?.is_mobile ? (
                      <Smartphone className="w-4 h-4 shrink-0 text-gray-400" />
                    ) : (
                      <Monitor className="w-4 h-4 shrink-0 text-gray-400" />
                    )}
                    <div className="min-w-0">
                      <p className="truncate text-sm text-gray-900">{deviceLabel(subscriber)}</p>
                      <p className="text-xs text-gray-500">
                        Joined {format(new Date(subscriber.created_at), 'dd MMM yyyy')}
                      </p>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      </div>

      <Card padding="none">
        <div className="border-b border-gray-200 px-6 py-4">
          <h2 className="font-semibold text-gray-900">Recent broadcasts</h2>
        </div>

        {broadcasts.length === 0 ? (
          <p className="p-12 text-center text-gray-500">Nothing sent yet.</p>
        ) : (
          <div className="divide-y divide-gray-200">
            {broadcasts.map((broadcast) => (
              <div key={broadcast.id} className="px-6 py-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <p className="font-medium text-gray-900">{broadcast.title}</p>
                    <p className="mt-0.5 truncate text-sm text-gray-600">{broadcast.body}</p>
                    <p className="mt-1 text-xs text-gray-500">
                      {format(new Date(broadcast.sent_at), 'dd MMM yyyy, HH:mm')} ·{' '}
                      {broadcast.success_count}/{broadcast.total_targeted} delivered
                    </p>
                  </div>
                  <Badge variant={statusVariant[broadcast.status]}>{broadcast.status}</Badge>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
