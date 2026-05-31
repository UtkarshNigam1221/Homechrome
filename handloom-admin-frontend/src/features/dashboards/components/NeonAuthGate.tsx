import { useEffect, useState } from 'react';

import { clearNeonAuthCache, neonAuthClient } from '@/shared/auth/neonAuth';

interface Props {
  children: React.ReactNode;
}

type GateState = 'loading' | 'authed' | 'unauthed';

// Path A dual-auth: dashboards check for an active Neon Auth session before
// rendering data (since Neon Data API needs a Neon Auth JWT — not the admin
// custom JWT). If no session, prompt the user to sign in with Google.
export function NeonAuthGate({ children }: Props) {
  const [state, setState] = useState<GateState>('loading');
  const [email, setEmail] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    void (async () => {
      try {
        const session = await neonAuthClient.getSession();
        if (!mounted) return;
        // Better Auth returns { data: { user, session } } or { data: null }
        const user = session?.data?.user ?? null;
        if (user) {
          setEmail(user.email ?? null);
          setState('authed');
        } else {
          setState('unauthed');
        }
      } catch {
        if (mounted) setState('unauthed');
      }
    })();
    return () => {
      mounted = false;
    };
  }, []);

  if (state === 'loading') {
    return (
      <div className="flex h-full items-center justify-center p-8">
        <p className="text-sm text-neutral-500">Checking Neon Auth session…</p>
      </div>
    );
  }

  if (state === 'unauthed') {
    return (
      <div className="flex h-full items-center justify-center p-8">
        <div className="max-w-md rounded-lg border border-neutral-200 bg-white p-6 shadow-sm">
          <h2 className="text-lg font-semibold text-neutral-900">Sign in to view dashboards</h2>
          <p className="mt-2 text-sm text-neutral-600">
            Dashboards use Neon Data API which requires a separate Neon Auth login (in addition to
            your admin login).
          </p>
          <button
            type="button"
            onClick={() => {
              void neonAuthClient.signIn.social({
                provider: 'google',
                callbackURL: window.location.href,
              });
            }}
            className="mt-4 inline-flex items-center gap-2 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700"
          >
            Sign in with Google
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-end gap-3 border-b border-neutral-200 bg-white px-6 py-1.5 text-xs text-neutral-500">
        {email && <span>Neon Auth: {email}</span>}
        <button
          type="button"
          onClick={() => {
            void (async () => {
              try {
                await neonAuthClient.signOut();
              } finally {
                clearNeonAuthCache();
                setState('unauthed');
                setEmail(null);
              }
            })();
          }}
          className="rounded px-2 py-0.5 text-neutral-600 hover:bg-neutral-100"
        >
          Sign out
        </button>
      </div>
      <div className="flex-1 overflow-auto">{children}</div>
    </div>
  );
}
