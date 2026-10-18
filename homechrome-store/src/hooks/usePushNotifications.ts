'use client';

import { useCallback, useEffect } from 'react';
import { create } from 'zustand';

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

// One in-flight fetch shared by every caller: the key is a per-deployment
// constant, and a failure clears the cache so the next caller retries.
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

// Subscribes with the current key, dropping a subscription issued for an
// older one first, and registers the device with the backend.
async function registerDevice(registration: ServiceWorkerRegistration, publicKey: string): Promise<void> {
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

  await apiClient.post(ROUTES.PUSH.SUBSCRIBE, { endpoint, keys, device: detectDeviceInfo() });
}

// Shared across every mounted instance, so subscribing in the modal is
// reflected by the banner without a remount.
const usePushStore = create<PushStatus>(() => ({ ...UNSUPPORTED, loading: true }));
const setSharedStatus = usePushStore.setState;

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
  const status = usePushStore();

  const refresh = useCallback(async () => {
    if (!pushSupported()) {
      setSharedStatus(UNSUPPORTED);
      return;
    }

    try {
      const registration = await navigator.serviceWorker.getRegistration();
      const subscription = registration ? await registration.pushManager.getSubscription() : null;

      // An unconfigured backend returns an empty key. Treating that as
      // "configured" would show an opt-in that can only ever fail.
      const publicKey = await fetchVapidKey();

      // A subscription bound to a superseded key is undeliverable and never
      // pruned (the push service answers 403, not 410), so it counts as none.
      let usable =
        !!subscription &&
        (!publicKey || usesKey(subscription, urlBase64ToUint8Array(publicKey)));

      // Permission is already granted, so no opt-in will return to repair a
      // stale key; re-register silently instead.
      if (subscription && publicKey && !usable && registration && Notification.permission === 'granted') {
        try {
          await registerDevice(registration, publicKey);
          usable = true;
        } catch {
          /* stays reported as unsubscribed; the modal toggle can retry */
        }
      }

      setSharedStatus({
        isSupported: true,
        permission: Notification.permission,
        isSubscribed: usable,
        isConfigured: !!publicKey,
        loading: false,
        error: null,
      });
    } catch (err) {
      setSharedStatus((prev) => ({
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
      setSharedStatus((prev) => ({
        ...prev,
        error: 'This browser does not support push notifications.',
      }));
      return false;
    }

    setSharedStatus((prev) => ({ ...prev, loading: true, error: null }));

    try {
      const permission = await Notification.requestPermission();
      if (permission !== 'granted') {
        setSharedStatus((prev) => ({
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

      await registerDevice(registration, publicKey);

      setSharedStatus({
        isSupported: true,
        permission: 'granted',
        isSubscribed: true,
        isConfigured: true,
        loading: false,
        error: null,
      });
      return true;
    } catch (err) {
      setSharedStatus((prev) => ({
        ...prev,
        loading: false,
        error: errorMessage(err, 'Could not enable notifications'),
      }));
      return false;
    }
  }, []);

  const unsubscribe = useCallback(async (): Promise<boolean> => {
    setSharedStatus((prev) => ({ ...prev, loading: true, error: null }));

    try {
      const registration = await navigator.serviceWorker.getRegistration();
      const subscription = registration ? await registration.pushManager.getSubscription() : null;

      if (subscription) {
        const { endpoint } = subscription;
        await subscription.unsubscribe();
        // The browser has let go, so report that now; a failed backend call
        // leaves a stale row, which the next 410 prunes.
        setSharedStatus((prev) => ({ ...prev, isSubscribed: false }));
        await apiClient.post(ROUTES.PUSH.UNSUBSCRIBE, { endpoint });
      }

      setSharedStatus((prev) => ({ ...prev, isSubscribed: false, loading: false }));
      return true;
    } catch (err) {
      setSharedStatus((prev) => ({
        ...prev,
        loading: false,
        error: errorMessage(err, 'Could not turn off notifications'),
      }));
      return false;
    }
  }, []);

  /** Sends to this device only; the backend route is scoped to the endpoint in the body. */
  const sendTest = useCallback(async (): Promise<boolean> => {
    const registration = await navigator.serviceWorker.getRegistration();
    const subscription = registration ? await registration.pushManager.getSubscription() : null;
    if (!subscription) {
      setSharedStatus((prev) => ({ ...prev, error: 'This device is not subscribed.' }));
      return false;
    }

    await apiClient.post(ROUTES.PUSH.TEST, { endpoint: subscription.endpoint });
    return true;
  }, []);

  return { ...status, subscribe, unsubscribe, sendTest, refresh };
}
