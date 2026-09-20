'use client';

import { useCallback, useEffect, useSyncExternalStore } from 'react';

import apiClient from '@/lib/api';
import { ROUTES } from '@/lib/routes';

export interface PushStatus {
  isSupported: boolean;
  permission: NotificationPermission | 'unsupported';
  isSubscribed: boolean;
  /** The backend has no VAPID key configured, so nothing can be subscribed. */
  isConfigured: boolean;
  loading: boolean;
  error: string | null;
}

const UNSUPPORTED: PushStatus = {
  isSupported: false,
  permission: 'unsupported',
  isSubscribed: false,
  isConfigured: false,
  loading: false,
  error: null,
};

/** PushManager wants the VAPID key as raw bytes, not the base64url string. */
function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

/** Whether a subscription was issued for this VAPID key. */
function usesKey(subscription: PushSubscription, key: Uint8Array): boolean {
  const current = subscription.options.applicationServerKey;
  if (!current) return false;
  const bytes = new Uint8Array(current);
  return bytes.length === key.length && bytes.every((b, i) => b === key[i]);
}

/** Coarse browser/OS label, so the admin console can tell devices apart. */
function detectDeviceInfo() {
  if (typeof window === 'undefined') return undefined;
  const ua = navigator.userAgent;

  // Order matters: Edge and Opera both carry "Chrome", Chrome carries "Safari".
  let browser = 'Unknown';
  if (ua.includes('Firefox')) browser = 'Firefox';
  else if (ua.includes('SamsungBrowser')) browser = 'Samsung Internet';
  else if (ua.includes('OPR') || ua.includes('Opera')) browser = 'Opera';
  else if (ua.includes('Edg')) browser = 'Edge';
  else if (ua.includes('Chrome')) browser = 'Chrome';
  else if (ua.includes('Safari')) browser = 'Safari';

  let os = 'Unknown';
  if (/iPad|iPhone|iPod/.test(ua)) os = 'iOS';
  else if (/Android/.test(ua)) os = 'Android';
  else if (/Macintosh|Mac OS X/.test(ua)) os = 'macOS';
  else if (/Windows/.test(ua)) os = 'Windows';
  else if (/Linux/.test(ua)) os = 'Linux';

  return { browser, os, is_mobile: /Mobi|Android|iPhone|iPad|iPod/i.test(ua) };
}

/**
 * The VAPID key is a per-deployment constant, but both the opt-in banner and
 * the settings modal mount on every page. Without this, each page view costs
 * two identical round trips. One in-flight promise is shared by all callers;
 * a failure clears it so the next caller retries.
 */
let vapidKeyPromise: Promise<string> | null = null;

function fetchVapidKey(): Promise<string> {
  vapidKeyPromise ??= apiClient
    .get<{ public_key: string }>(ROUTES.PUSH.VAPID_KEY)
    .then(({ data }) => data?.public_key ?? '')
    .catch((err: unknown) => {
      vapidKeyPromise = null;
      throw err;
    });
  return vapidKeyPromise;
}

/** This browser's live push subscription, or null when there is none. */
async function currentSubscription(): Promise<PushSubscription | null> {
  const registration = await navigator.serviceWorker.getRegistration();
  return registration ? registration.pushManager.getSubscription() : null;
}

function pushSupported(): boolean {
  return (
    typeof window !== 'undefined' &&
    'serviceWorker' in navigator &&
    'PushManager' in window &&
    'Notification' in window
  );
}

function errorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback;
}

// Shared across every mounted instance. The opt-in banner and the header modal
// each call this hook, and with per-instance state, subscribing in one left the
// other still advertising the opt-in until it remounted.
let sharedStatus: PushStatus = { ...UNSUPPORTED, loading: true };
const statusListeners = new Set<() => void>();

function setSharedStatus(next: PushStatus | ((prev: PushStatus) => PushStatus)) {
  sharedStatus = typeof next === 'function' ? next(sharedStatus) : next;
  statusListeners.forEach((notify) => notify());
}

function subscribeToStatus(listener: () => void) {
  statusListeners.add(listener);
  return () => {
    statusListeners.delete(listener);
  };
}

const readStatus = () => sharedStatus;

/**
 * Attach this browser's existing subscription to the shopper who just signed
 * in. Most people allow notifications before logging in, so without this their
 * device is never linked to their orders.
 */
export async function linkPushSubscription(): Promise<void> {
  if (!pushSupported()) return;
  try {
    const subscription = await currentSubscription();
    if (!subscription) return;

    // A subscription bound to a superseded key answers 403, not 410, so it is
    // never pruned — linking one only points the shopper at a dead device.
    const publicKey = await fetchVapidKey();
    if (publicKey && !usesKey(subscription, urlBase64ToUint8Array(publicKey))) return;

    await apiClient.post(ROUTES.PUSH.LINK, { endpoint: subscription.endpoint });
  } catch {
    // Linking is an optimisation: a shopper who misses it simply gets no order
    // pushes on this device until they next re-subscribe.
  }
}

/**
 * Detach this browser's subscription from the shopper signing out. Without it
 * customer_id outlives the session, and their next order update lands on a
 * lock screen whoever picks the device up next is holding.
 */
export async function unlinkPushSubscription(): Promise<void> {
  if (!pushSupported()) return;
  try {
    const subscription = await currentSubscription();
    if (!subscription) return;
    await apiClient.post(ROUTES.PUSH.UNLINK, { endpoint: subscription.endpoint });
  } catch {
    // Signing out must never be blocked by this; the device keeps its link
    // until a later sign-in hands it over or it re-subscribes.
  }
}

export function usePushNotifications() {
  const status = useSyncExternalStore(subscribeToStatus, readStatus, readStatus);
  const setStatus = setSharedStatus;

  const refresh = useCallback(async () => {
    if (!pushSupported()) {
      setStatus(UNSUPPORTED);
      return;
    }

    try {
      const registration = await navigator.serviceWorker.getRegistration();
      const subscription = registration ? await registration.pushManager.getSubscription() : null;

      // An unconfigured backend returns an empty key. Treating that as
      // "configured" would show an opt-in that can only ever fail.
      const publicKey = await fetchVapidKey();

      // A subscription bound to a superseded key is reported as no
      // subscription at all: the push service answers 403 for it rather than
      // 410, so it is never pruned, and calling it subscribed would leave the
      // device silently undeliverable with no opt-in to repair it.
      const usable =
        !!subscription &&
        (!publicKey || usesKey(subscription, urlBase64ToUint8Array(publicKey)));

      setStatus({
        isSupported: true,
        permission: Notification.permission,
        isSubscribed: usable,
        isConfigured: !!publicKey,
        loading: false,
        error: null,
      });
    } catch (err) {
      setStatus((prev) => ({
        ...prev,
        isSupported: true,
        loading: false,
        error: errorMessage(err, 'Could not check notification status'),
      }));
    }
  }, []);

  useEffect(() => {
    void refresh();

    // Permission can change from browser UI with no event on our side, which
    // otherwise leaves the opt-in offering something the browser will refuse.
    let permissionStatus: PermissionStatus | null = null;
    const onChange = () => void refresh();

    void navigator.permissions
      ?.query({ name: 'notifications' as PermissionName })
      .then((result) => {
        permissionStatus = result;
        result.addEventListener('change', onChange);
      })
      .catch(() => {
        // Safari historically rejects this query; the initial refresh stands.
      });

    return () => permissionStatus?.removeEventListener('change', onChange);
  }, [refresh]);

  const subscribe = useCallback(async (): Promise<boolean> => {
    if (!pushSupported()) {
      setStatus((prev) => ({
        ...prev,
        error: 'This browser does not support push notifications.',
      }));
      return false;
    }

    setStatus((prev) => ({ ...prev, loading: true, error: null }));

    try {
      const permission = await Notification.requestPermission();
      if (permission !== 'granted') {
        setStatus((prev) => ({
          ...prev,
          permission,
          isSubscribed: false,
          loading: false,
          error:
            permission === 'denied'
              ? 'Notifications are blocked in your browser settings.'
              : null,
        }));
        return false;
      }

      const registration =
        (await navigator.serviceWorker.getRegistration()) ??
        (await navigator.serviceWorker.register('/sw.js', { scope: '/' }));
      await navigator.serviceWorker.ready;

      const publicKey = await fetchVapidKey();
      if (!publicKey) {
        throw new Error('Push notifications are not available right now.');
      }

      // Reuse an existing subscription only when it was issued for the current
      // key. One bound to a rotated key can never be delivered to, and the push
      // service answers 403 rather than 410, so it is never pruned either.
      const applicationServerKey = urlBase64ToUint8Array(publicKey);
      let existing = await registration.pushManager.getSubscription();
      if (existing && !usesKey(existing, applicationServerKey)) {
        await existing.unsubscribe();
        existing = null;
      }

      const subscription =
        existing ??
        (await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: applicationServerKey as BufferSource,
        }));

      const { endpoint, keys } = subscription.toJSON() as {
        endpoint: string;
        keys: { p256dh: string; auth: string };
      };

      await apiClient.post(ROUTES.PUSH.SUBSCRIBE, {
        endpoint,
        keys,
        device: detectDeviceInfo(),
      });

      setStatus({
        isSupported: true,
        permission: 'granted',
        isSubscribed: true,
        isConfigured: true,
        loading: false,
        error: null,
      });
      return true;
    } catch (err) {
      setStatus((prev) => ({
        ...prev,
        loading: false,
        error: errorMessage(err, 'Could not enable notifications'),
      }));
      return false;
    }
  }, []);

  const unsubscribe = useCallback(async (): Promise<boolean> => {
    setStatus((prev) => ({ ...prev, loading: true, error: null }));

    try {
      const registration = await navigator.serviceWorker.getRegistration();
      const subscription = registration ? await registration.pushManager.getSubscription() : null;

      if (subscription) {
        const { endpoint } = subscription;
        await subscription.unsubscribe();
        // Tell the backend after the browser has let go, so a failed request
        // leaves a stale row rather than a live endpoint we no longer track.
        await apiClient.post(ROUTES.PUSH.UNSUBSCRIBE, { endpoint });
      }

      setStatus((prev) => ({ ...prev, isSubscribed: false, loading: false }));
      return true;
    } catch (err) {
      setStatus((prev) => ({
        ...prev,
        loading: false,
        error: errorMessage(err, 'Could not turn off notifications'),
      }));
      return false;
    }
  }, []);

  /**
   * Sends a notification to THIS device only. The backend route is scoped to
   * the endpoint in the body, so this can never reach another visitor.
   */
  const sendTest = useCallback(async (): Promise<boolean> => {
    const registration = await navigator.serviceWorker.getRegistration();
    const subscription = registration ? await registration.pushManager.getSubscription() : null;
    if (!subscription) {
      setStatus((prev) => ({ ...prev, error: 'This device is not subscribed.' }));
      return false;
    }

    await apiClient.post(ROUTES.PUSH.TEST, { endpoint: subscription.endpoint });
    return true;
  }, []);

  return { ...status, subscribe, unsubscribe, sendTest, refresh };
}
