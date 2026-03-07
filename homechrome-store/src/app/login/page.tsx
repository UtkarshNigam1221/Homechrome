'use client';

import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useCallback, useEffect, useRef, useState } from 'react';

import Button from '@/components/common/Button';
import Input from '@/components/common/Input';
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

  // If already authenticated, redirect
  useEffect(() => {
    if (isAuthenticated) {
      router.replace(redirect);
    }
  }, [isAuthenticated, redirect, router]);

  // Countdown timer — start interval only once, let it self-clear at 0
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
    <div className="flex min-h-[60vh] items-center justify-center px-4 py-16">
      <div className="w-full max-w-sm">
        {/* Logo */}
        <div className="mb-8 text-center">
          <Link href="/">
            <span className="text-2xl font-bold tracking-tight text-foreground">
              HOME<span className="text-primary">CHROME</span>
            </span>
          </Link>
          <p className="mt-3 text-sm text-muted">
            Sign in to your account
          </p>
        </div>

        <div className="rounded-xl bg-white p-6 shadow-sm">
          {step === 'phone' ? (
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleSendOTP();
              }}
            >
              <div>
                <label
                  htmlFor="phone"
                  className="mb-1.5 block text-sm font-medium text-foreground"
                >
                  Phone Number
                </label>
                <div className="flex gap-2">
                  <div className="flex items-center rounded-lg border border-border bg-gray-50 px-3 text-sm text-muted">
                    +91
                  </div>
                  <Input
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
                </div>
              </div>

              <Button
                type="submit"
                variant="primary"
                size="md"
                className="mt-4 w-full"
                loading={sending}
                disabled={phone.replace(/\D/g, '').length !== 10}
              >
                Send OTP
              </Button>
            </form>
          ) : (
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleVerifyOTP();
              }}
            >
              <p className="mb-4 text-sm text-muted">
                We sent a 6-digit code to{' '}
                <span className="font-medium text-foreground">
                  +91 {phone}
                </span>
              </p>

              <Input
                id="otp"
                label="Enter OTP"
                type="text"
                inputMode="numeric"
                placeholder="000000"
                value={otp}
                onChange={(e) => {
                  setOtp(e.target.value.replace(/\D/g, '').slice(0, 6));
                  setError('');
                }}
                maxLength={6}
                error={error || undefined}
              />

              <Button
                type="submit"
                variant="primary"
                size="md"
                className="mt-4 w-full"
                loading={verifying}
                disabled={otp.length !== 6}
              >
                Verify OTP
              </Button>

              <div className="mt-4 text-center">
                {countdown > 0 ? (
                  <p className="text-sm text-muted">
                    Resend OTP in{' '}
                    <span className="font-medium text-foreground">{countdown}s</span>
                  </p>
                ) : (
                  <button
                    type="button"
                    onClick={handleResendOTP}
                    disabled={sending}
                    className="text-sm text-primary hover:text-primary-dark disabled:opacity-50"
                  >
                    Resend OTP
                  </button>
                )}
              </div>

              <button
                type="button"
                onClick={() => {
                  setStep('phone');
                  setOtp('');
                  setError('');
                }}
                className="mt-3 w-full text-center text-sm text-muted hover:text-foreground"
              >
                Change phone number
              </button>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-[60vh] items-center justify-center">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      }
    >
      <LoginForm />
    </Suspense>
  );
}
