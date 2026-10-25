'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState, type SubmitEvent } from 'react';
import { Button } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import { TextField } from '@/components/ui/text-field';
import { api, errorMessage, UNREACHABLE } from '@/lib/api/client';
import {
  PASSWORD_MAX_LENGTH,
  PASSWORD_MIN_LENGTH,
  USERNAME_MAX_LENGTH,
  USERNAME_MIN_LENGTH,
  USERNAME_PATTERN,
} from '@/lib/credentials';

interface AuthFormProps {
  readonly mode: 'login' | 'signup';
  /** Where to go once logged in, already checked to be a page of this site. */
  readonly next: string;
}

const COPY = {
  login: {
    title: 'Log in',
    submit: 'Log in',
    pending: 'Logging in…',
    switchText: 'New here?',
    switchLink: 'Create an account',
    switchHref: '/signup',
  },
  signup: {
    title: 'Create an account',
    submit: 'Sign up',
    pending: 'Creating account…',
    switchText: 'Already have an account?',
    switchLink: 'Log in',
    switchHref: '/login',
  },
} as const;

/** The sign-up and log-in form. The Go server checks everything and sets the session cookie. */
export function AuthForm({ mode, next }: AuthFormProps) {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const copy = COPY[mode];
  const isSignup = mode === 'signup';
  const switchHref =
    next === '/' ? copy.switchHref : `${copy.switchHref}?next=${encodeURIComponent(next)}`;

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const body = { username: textField(form, 'username'), password: textField(form, 'password') };

    setPending(true);
    setError(null);
    try {
      const { error: failure, response } = isSignup
        ? await api.POST('/auth/signup', { body })
        : await api.POST('/auth/login', { body });

      if (response.ok) {
        router.push(next);
        router.refresh();
        return;
      }
      setError(errorMessage(failure));
    } catch {
      setError(UNREACHABLE);
    } finally {
      setPending(false);
    }
  }

  return (
    <Panel accent="water" className="mx-auto w-full max-w-sm p-6">
      <h1 className="font-display text-3xl font-bold tracking-widest text-white uppercase">
        {copy.title}
      </h1>

      <form onSubmit={(event) => void submit(event)} className="mt-6 space-y-4">
        <TextField
          label="Username"
          name="username"
          autoComplete="username"
          required
          minLength={USERNAME_MIN_LENGTH}
          maxLength={USERNAME_MAX_LENGTH}
          pattern={USERNAME_PATTERN}
          hint={
            isSignup
              ? `${String(USERNAME_MIN_LENGTH)}–${String(USERNAME_MAX_LENGTH)} letters, numbers, _ or -`
              : undefined
          }
        />
        <TextField
          label="Password"
          name="password"
          type="password"
          autoComplete={isSignup ? 'new-password' : 'current-password'}
          required
          minLength={PASSWORD_MIN_LENGTH}
          maxLength={PASSWORD_MAX_LENGTH}
          hint={isSignup ? `At least ${String(PASSWORD_MIN_LENGTH)} characters` : undefined}
        />

        {error && (
          <p
            role="alert"
            className="bg-red-500/10 px-3 py-2 text-sm text-red-200 ring-1 ring-red-400/40 ring-inset"
          >
            {error}
          </p>
        )}

        <Button type="submit" className="w-full" disabled={pending}>
          {pending ? copy.pending : copy.submit}
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-slate-400">
        {copy.switchText}{' '}
        <Link
          href={switchHref}
          className="font-semibold text-accent underline-offset-4 hover:underline"
        >
          {copy.switchLink}
        </Link>
      </p>
    </Panel>
  );
}

/** A text box's value; empty if the form has no such text box. */
function textField(form: FormData, name: string): string {
  const value = form.get(name);
  return typeof value === 'string' ? value : '';
}
