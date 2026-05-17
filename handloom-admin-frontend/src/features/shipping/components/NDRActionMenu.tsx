import { Menu, Transition } from '@headlessui/react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { clsx } from 'clsx';
import { MoreHorizontal, PhoneCall, RotateCcw, Undo2 } from 'lucide-react';
import { Fragment } from 'react';
import toast from 'react-hot-toast';

import { shippingApi } from '@/features/shipping/api';
import type { NDRAction } from '@/features/shipping/types';
import { getErrorMessage } from '@/shared/api/client';
import { Button } from '@/shared/components/ui';

const ACTIONS: {
  action: NDRAction;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
}[] = [
  { action: 'reattempt', label: 'Re-attempt delivery', icon: RotateCcw },
  { action: 'mark_contacted', label: 'Mark customer contacted', icon: PhoneCall },
  { action: 'rto', label: 'Return to origin (RTO)', icon: Undo2 },
];

interface NDRActionMenuProps {
  awb: string;
}

export function NDRActionMenu({ awb }: NDRActionMenuProps) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (action: NDRAction) => shippingApi.ndrAction(awb, action),
    onSuccess: () => {
      toast.success('Action queued');
      queryClient.invalidateQueries({ queryKey: ['shipping', 'ndr-queue'] });
    },
    onError: (error) => {
      toast.error(getErrorMessage(error) || 'NDR action failed');
    },
  });

  return (
    <Menu as="div" className="relative inline-block text-left">
      <Menu.Button as={Fragment}>
        <Button
          variant="ghost"
          size="sm"
          title="NDR actions"
          aria-label="NDR actions"
          loading={mutation.isPending}
        >
          <MoreHorizontal className="w-4 h-4" />
        </Button>
      </Menu.Button>
      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <Menu.Items className="absolute right-0 z-20 mt-1 w-56 origin-top-right rounded-lg bg-white shadow-lg ring-1 ring-black/5 focus:outline-none">
          <div className="py-1">
            {ACTIONS.map(({ action, label, icon: Icon }) => (
              <Menu.Item key={action} disabled={mutation.isPending}>
                {({ active, disabled }) => (
                  <button
                    type="button"
                    onClick={() => mutation.mutate(action)}
                    disabled={disabled}
                    className={clsx(
                      'flex w-full items-center gap-2 px-3 py-2 text-left text-sm',
                      active ? 'bg-gray-100 text-gray-900' : 'text-gray-700',
                      disabled && 'opacity-50 cursor-not-allowed'
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    {label}
                  </button>
                )}
              </Menu.Item>
            ))}
          </div>
        </Menu.Items>
      </Transition>
    </Menu>
  );
}
