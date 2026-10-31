'use client';

import { useRouter } from 'next/navigation';
import { useEffect, useState, type ReactNode, type SubmitEvent } from 'react';
import { LogOutButton } from '@/components/log-out-button';
import { Button } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import { TextField } from '@/components/ui/text-field';
import { changeVerificationEmail, confirmEmail, sendVerificationCode } from '@/lib/account-actions';
import type { Verification } from '@/lib/api/client';
import type { Outcome } from '@/lib/api/outcome';
import { textField } from '@/lib/form-data';
import { notify } from '@/lib/notifications';
import { CODE_DIGITS, EMAIL_MAX_LENGTH, verificationStep, waitText } from '@/lib/verification';

/** How often a waiting player checks whether they can go on, by mail or by an admin letting them in. */
const CHECK_WAITING_EVERY_MS = 30_000;

interface VerifyEmailFormProps {
  readonly status: Verification;
  /** Where to go once confirmed, already checked to be a page of this site. */
  readonly next: string;
}

/**
 * Confirming an email address: enter the code sent, ask for another, or correct the address. The
 * page draws this again from the server after each step, so it always shows how things stand.
 */
export function VerifyEmailForm({ status, next }: VerifyEmailFormProps) {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [changing, setChanging] = useState(false);
  const step = verificationStep(status);
  const waiting = step === 'waiting';
  // An error belongs with whichever form is open.
  const stepError = changing ? null : error;

  useEffect(() => {
    if (!waiting) return;

    const timer = setInterval(() => {
      router.refresh();
    }, CHECK_WAITING_EVERY_MS);
    return () => {
      clearInterval(timer);
    };
  }, [waiting, router]);

  async function run<T>(send: () => Promise<Outcome<T>>, done: (value: T) => void) {
    setPending(true);
    setError(null);
    const outcome = await send();
    setPending(false);
    if (!outcome.ok) {
      setError(outcome.message);
      return;
    }
    done(outcome.value);
  }

  function submitCode(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const code = textField(new FormData(event.currentTarget), 'code');
    void run(
      () => confirmEmail(code),
      () => {
        notify('Your email address is confirmed. Welcome!');
        router.push(next);
        router.refresh();
      },
    );
  }

  function sendCode() {
    void run(sendVerificationCode, () => {
      notify(`A code is on its way to ${status.email}.`);
      router.refresh();
    });
  }

  function submitEmail(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const email = textField(new FormData(event.currentTarget), 'email');
    void run(
      () => changeVerificationEmail(email),
      () => {
        setChanging(false);
        router.refresh();
      },
    );
  }

  return (
    <Panel accent="water" className="mx-auto w-full max-w-md p-6">
      <h1 className="font-display text-3xl font-bold tracking-widest text-white uppercase">
        Confirm your email
      </h1>

      {step === 'enter' && (
        <>
          <Lead>
            We sent a {CODE_DIGITS}-digit code to <Address>{status.email}</Address>. Check your spam
            folder if you cannot see it.
          </Lead>
          <form onSubmit={submitCode} className="mt-6 space-y-4">
            <TextField
              label="Code"
              name="code"
              required
              inputMode="numeric"
              autoComplete="one-time-code"
              autoFocus
              maxLength={CODE_DIGITS + 2}
              hint="An account not confirmed in time is deleted, which frees its name and address."
            />
            <ErrorMessage error={stepError} />
            <Button type="submit" className="w-full" disabled={pending}>
              {pending ? 'Checking…' : 'Confirm'}
            </Button>
          </form>
          {status.mailWorks && (
            <ResendButton
              key={status.codeSentAt}
              seconds={status.resendInSeconds}
              disabled={pending}
              onClick={sendCode}
            />
          )}
        </>
      )}

      {step === 'send' && (
        <>
          <Lead>
            We need to send a code to <Address>{status.email}</Address>.
          </Lead>
          <div className="mt-6 space-y-4">
            <ErrorMessage error={stepError} />
            <Button className="w-full" disabled={pending} onClick={sendCode}>
              {pending ? 'Sending…' : 'Send the code'}
            </Button>
          </div>
        </>
      )}

      {step === 'waiting' && (
        <>
          <Lead>
            Your account is ready, but the site cannot send email yet, so we cannot send a code to{' '}
            <Address>{status.email}</Address>. An admin can let you in, or you can come back later.
            This page checks by itself.
          </Lead>
          <ErrorMessage error={stepError} className="mt-4" />
        </>
      )}

      <div className="mt-6 border-t border-slate-700/60 pt-4">
        {changing ? (
          <form onSubmit={submitEmail} className="space-y-3">
            <TextField
              label="New email address"
              name="email"
              type="email"
              required
              autoComplete="email"
              maxLength={EMAIL_MAX_LENGTH}
              defaultValue={status.email}
            />
            <ErrorMessage error={error} />
            <div className="flex gap-2">
              <Button type="submit" disabled={pending}>
                Save
              </Button>
              <Button
                variant="quiet"
                onClick={() => {
                  setChanging(false);
                  setError(null);
                }}
              >
                Cancel
              </Button>
            </div>
          </form>
        ) : (
          <div className="flex items-center justify-between gap-4">
            <button
              type="button"
              className="cursor-pointer text-sm font-semibold text-accent underline-offset-4 hover:underline"
              onClick={() => {
                setChanging(true);
                setError(null);
              }}
            >
              Wrong address?
            </button>
            <LogOutButton />
          </div>
        )}
      </div>
    </Panel>
  );
}

function Lead({ children }: { readonly children: ReactNode }) {
  return <p className="mt-4 text-sm text-slate-300">{children}</p>;
}

function Address({ children }: { readonly children: ReactNode }) {
  return <strong className="break-all text-white">{children}</strong>;
}

function ErrorMessage({
  error,
  className = '',
}: {
  readonly error: string | null;
  readonly className?: string;
}) {
  if (!error) return null;

  return (
    <p
      role="alert"
      className={`bg-red-500/10 px-3 py-2 text-sm text-red-200 ring-1 ring-red-400/40 ring-inset ${className}`}
    >
      {error}
    </p>
  );
}

/** Asks for another code, once the wait since the last one is over. */
function ResendButton({
  seconds,
  disabled,
  onClick,
}: {
  readonly seconds: number;
  readonly disabled: boolean;
  readonly onClick: () => void;
}) {
  const [left, setLeft] = useState(seconds);
  const waiting = left > 0;

  useEffect(() => {
    if (!waiting) return;

    const timer = setInterval(() => {
      setLeft((current) => Math.max(0, current - 1));
    }, 1000);
    return () => {
      clearInterval(timer);
    };
  }, [waiting]);

  return (
    <Button
      variant="secondary"
      className="mt-3 w-full"
      disabled={disabled || waiting}
      onClick={onClick}
    >
      {waiting ? `Send another code in ${waitText(left)}` : 'Send another code'}
    </Button>
  );
}
