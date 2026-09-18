import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { Megaphone, Monitor, Send, Smartphone, Users } from 'lucide-react';
import { useState } from 'react';
import toast from 'react-hot-toast';

import { pushApi } from '@/features/push/api';
import { NotificationPreview } from '@/features/push/components/NotificationPreview';
import { PUSH_TEMPLATES } from '@/features/push/templates';
import { getErrorMessage } from '@/shared/api/client';
import { PageLoading } from '@/shared/components/loading';
import { Badge, Button, Card, Input, PageHeader } from '@/shared/components/ui';

import type { PushAction, PushBroadcastStatus, PushSubscriber } from '../types';

const TITLE_MAX = 120;
const BODY_MAX = 300;
// Android truncates past these; the hard caps above are the backend's.
const TITLE_COMFORTABLE = 40;
const BODY_COMFORTABLE = 100;
const ACTION_TITLE_MAX = 24;

const EMPTY_ACTION: PushAction = { action: '', title: '', url: '' };

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
  const [icon, setIcon] = useState('');
  const [actions, setActions] = useState<PushAction[]>([{ ...EMPTY_ACTION }]);

  const applyTemplate = (id: string) => {
    const template = PUSH_TEMPLATES.find((t) => t.id === id);
    if (!template) return;
    setTitle(template.values.title);
    setBody(template.values.body);
    setUrl(template.values.url);
    setImage(template.values.image ?? '');
    setIcon(template.values.icon ?? '');
    setActions(
      template.values.actions?.length
        ? template.values.actions.map((a) => ({ ...a }))
        : [{ ...EMPTY_ACTION }]
    );
  };

  const updateAction = (index: number, patch: Partial<PushAction>) => {
    setActions((prev) => prev.map((a, i) => (i === index ? { ...a, ...patch } : a)));
  };

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
      setIcon('');
      setActions([{ ...EMPTY_ACTION }]);
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

  // A button needs a label; the id is derived so operators never type one.
  const preparedActions: PushAction[] = actions
    .filter((a) => a.title.trim())
    .slice(0, 2)
    .map((a, i) => ({
      action: a.action.trim() || `action_${i + 1}`,
      title: a.title.trim(),
      url: a.url?.trim() || undefined,
    }));

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSend) return;
    broadcastMutation.mutate({
      title: title.trim(),
      body: body.trim(),
      url: url.trim() || '/',
      tag: tag.trim() || undefined,
      image: image.trim() || undefined,
      icon: icon.trim() || undefined,
      actions: preparedActions.length ? preparedActions : undefined,
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

            <div>
              <p className="mb-2 text-sm font-medium text-gray-700">Start from</p>
              <div className="flex flex-wrap gap-2">
                {PUSH_TEMPLATES.map((template) => (
                  <button
                    key={template.id}
                    type="button"
                    onClick={() => applyTemplate(template.id)}
                    title={template.purpose}
                    className="rounded-full border border-gray-300 px-3 py-1.5 text-sm text-gray-700 transition-colors hover:border-primary-400 hover:bg-primary-50 hover:text-primary-800 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  >
                    {template.label}
                  </button>
                ))}
              </div>
            </div>

            <Input
              label="Title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              maxLength={TITLE_MAX}
              placeholder="Festive Handloom Drop — 20% off"
              hint={
                title.length > TITLE_COMFORTABLE
                  ? `${title.length}/${TITLE_MAX} — over ${TITLE_COMFORTABLE}, Android truncates on most phones`
                  : `${title.length}/${TITLE_MAX}`
              }
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
                {body.length > BODY_COMFORTABLE &&
                  ` — past ${BODY_COMFORTABLE} only shows when expanded`}
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

            <Input
              label="Icon override (optional)"
              value={icon}
              onChange={(e) => setIcon(e.target.value)}
              placeholder="https://dev-store.homechrome.in/products/dohar.png"
              hint="Square product thumbnail shown instead of the brand mark. 192×192."
            />

            <div className="border-t border-gray-200 pt-4">
              <div className="mb-1 flex items-baseline justify-between">
                <p className="text-sm font-medium text-gray-700">Buttons</p>
                <p className="text-xs text-gray-500">Android shows at most two</p>
              </div>
              <p className="mb-3 text-xs text-gray-500">
                Only visible when the notification is expanded. Leave the label empty to omit.
              </p>

              <div className="space-y-3">
                {[0, 1].map((index) => (
                  <div key={index} className="grid gap-3 sm:grid-cols-2">
                    <Input
                      label={`Button ${index + 1} label`}
                      value={actions[index]?.title ?? ''}
                      maxLength={ACTION_TITLE_MAX}
                      onChange={(e) => {
                        if (!actions[index]) {
                          setActions((prev) => [
                            ...prev,
                            { ...EMPTY_ACTION, title: e.target.value },
                          ]);
                          return;
                        }
                        updateAction(index, { title: e.target.value });
                      }}
                      placeholder={index === 0 ? 'Shop the loom' : 'Remind me'}
                    />
                    <Input
                      label={`Button ${index + 1} link`}
                      value={actions[index]?.url ?? ''}
                      onChange={(e) => {
                        if (!actions[index]) return;
                        updateAction(index, { url: e.target.value });
                      }}
                      disabled={!actions[index]?.title}
                      placeholder="/products"
                      hint={index === 0 ? 'Defaults to the click-through URL' : undefined}
                    />
                  </div>
                ))}
              </div>
            </div>

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
            <NotificationPreview
              title={title}
              body={body}
              image={image.trim() || undefined}
              icon={icon.trim() || undefined}
              actions={preparedActions}
            />
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
