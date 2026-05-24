'use client';

import {
  Anchor,
  Box,
  Button,
  Card,
  Center,
  Group,
  PinInput,
  Stack,
  Text,
  TextInput,
} from '@mantine/core';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useRef, useState } from 'react';

import { LoadingSpinner } from '@/components/ui/loading-spinner';
import { useAuthStore } from '@/stores/auth';

const OTP_RESEND_SECONDS = 30;

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirect = searchParams.get('redirect') || '/';

  const { sendOTP, verifyOTP, isAuthenticated } = useAuthStore();

  const [phone, setPhone] = useState('');
  const [otp, setOtp] = useState('');
  const [step, setStep] = useState<'phone' | 'otp'>('phone');
  const [sending, setSending] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [error, setError] = useState('');
  const [countdown, setCountdown] = useState(0);
  const countdownRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    if (isAuthenticated) {
      router.replace(redirect);
    }
  }, [isAuthenticated, redirect, router]);

  const startCountdown = useCallback((seconds: number) => {
    setCountdown(seconds);
    if (countdownRef.current) clearInterval(countdownRef.current);
    countdownRef.current = setInterval(() => {
      setCountdown((prev) => {
        if (prev <= 1) {
          if (countdownRef.current) clearInterval(countdownRef.current);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  }, []);

  useEffect(() => {
    return () => {
      if (countdownRef.current) clearInterval(countdownRef.current);
    };
  }, []);

  const handleSendOTP = useCallback(async () => {
    const cleaned = phone.replace(/\D/g, '');
    if (cleaned.length !== 10) {
      setError('Please enter a valid 10-digit phone number.');
      return;
    }
    setError('');
    setSending(true);
    try {
      await sendOTP(`+91${cleaned}`);
      setStep('otp');
      startCountdown(OTP_RESEND_SECONDS);
    } catch {
      setError('Failed to send OTP. Please try again.');
    } finally {
      setSending(false);
    }
  }, [phone, sendOTP, startCountdown]);

  const handleVerifyOTP = useCallback(async () => {
    if (otp.length !== 6) {
      setError('Please enter a valid 6-digit OTP.');
      return;
    }
    setError('');
    setVerifying(true);
    const cleaned = phone.replace(/\D/g, '');
    try {
      await verifyOTP(`+91${cleaned}`, otp);
      router.replace(redirect);
    } catch {
      setError('Invalid OTP. Please try again.');
    } finally {
      setVerifying(false);
    }
  }, [otp, phone, verifyOTP, router, redirect]);

  const handleResendOTP = useCallback(async () => {
    if (countdown > 0) return;
    setError('');
    setSending(true);
    const cleaned = phone.replace(/\D/g, '');
    try {
      await sendOTP(`+91${cleaned}`);
      startCountdown(OTP_RESEND_SECONDS);
    } catch {
      setError('Failed to resend OTP. Please try again.');
    } finally {
      setSending(false);
    }
  }, [countdown, phone, sendOTP, startCountdown]);

  return (
    <Center mih="60vh" px="md" py="xl">
      <Stack w="100%" maw={360} gap="lg">
        <Stack gap="xs" align="center">
          <Anchor component={Link} href="/" underline="never">
            <Text size="xl" fw={700} c="navy.7">
              HOME<Text span c="brand">CHROME</Text>
            </Text>
          </Anchor>
          <Text size="sm" c="dimmed">Sign in to your account</Text>
        </Stack>

        <Card shadow="sm" radius="lg" padding="lg">
          {step === 'phone' ? (
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleSendOTP();
              }}
            >
              <Stack gap="md">
                <Group gap="xs" align="flex-end" wrap="nowrap">
                  <Box
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      padding: '0 0.75rem',
                      height: 36,
                      borderRadius: 'var(--mantine-radius-md)',
                      border: '1px solid var(--mantine-color-default-border)',
                      background: 'var(--mantine-color-gray-1)',
                      color: 'var(--mantine-color-dimmed)',
                      fontSize: '0.875rem',
                    }}
                  >
                    +91
                  </Box>
                  <TextInput
                    flex={1}
                    label="Phone Number"
                    id="phone"
                    type="tel"
                    placeholder="10-digit number"
                    value={phone}
                    onChange={(e) => {
                      setPhone(e.target.value.replace(/\D/g, '').slice(0, 10));
                      setError('');
                    }}
                    maxLength={10}
                    error={error && step === 'phone' ? error : undefined}
                  />
                </Group>

                <Button
                  type="submit"
                  color="brand"
                  fullWidth
                  loading={sending}
                  disabled={phone.replace(/\D/g, '').length !== 10}
                >
                  Send OTP
                </Button>
              </Stack>
            </form>
          ) : (
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleVerifyOTP();
              }}
            >
              <Stack gap="md">
                <Text size="sm" c="dimmed">
                  We sent a 6-digit code to{' '}
                  <Text span fw={500} c="navy.7">+91 {phone}</Text>
                </Text>

                <Stack gap={4}>
                  <Text size="sm" fw={500} c="navy.7">Enter OTP</Text>
                  <PinInput
                    length={6}
                    type="number"
                    value={otp}
                    onChange={(value) => {
                      setOtp(value);
                      setError('');
                    }}
                    placeholder="•"
                    oneTimeCode
                    error={!!error}
                  />
                  {error && <Text size="xs" c="red.7">{error}</Text>}
                </Stack>

                <Button
                  type="submit"
                  color="brand"
                  fullWidth
                  loading={verifying}
                  disabled={otp.length !== 6}
                >
                  Verify OTP
                </Button>

                <Group justify="center">
                  {countdown > 0 ? (
                    <Text size="sm" c="dimmed">
                      Resend OTP in{' '}
                      <Text span fw={500} c="navy.7">{countdown}s</Text>
                    </Text>
                  ) : (
                    <Anchor
                      component="button"
                      type="button"
                      size="sm"
                      onClick={handleResendOTP}
                      disabled={sending}
                    >
                      Resend OTP
                    </Anchor>
                  )}
                </Group>

                <Button
                  variant="subtle"
                  color="navy"
                  size="sm"
                  fullWidth
                  onClick={() => {
                    setStep('phone');
                    setOtp('');
                    setError('');
                  }}
                >
                  Change phone number
                </Button>
              </Stack>
            </form>
          )}
        </Card>
      </Stack>
    </Center>
  );
}

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <Center mih="60vh">
          <LoadingSpinner size="lg" />
        </Center>
      }
    >
      <LoginForm />
    </Suspense>
  );
}
