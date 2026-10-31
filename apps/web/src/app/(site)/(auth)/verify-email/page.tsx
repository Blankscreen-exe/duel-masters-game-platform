import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { serverApi } from '@/lib/api/server';
import { safeRedirect } from '@/lib/safe-redirect';
import { getSessionAccount } from '@/lib/session';
import { VERIFY_EMAIL_PATH } from '@/lib/verification';
import { VerifyEmailForm } from './verify-email-form';

export const metadata: Metadata = { title: 'Confirm your email' };

/** Where a new player confirms their email address, the only thing they can do until then. */
export default async function VerifyEmailPage({ searchParams }: PageProps<'/verify-email'>) {
  const next = safeRedirect((await searchParams)['next']);

  const account = await getSessionAccount();
  if (!account) redirect(`/login?next=${encodeURIComponent(VERIFY_EMAIL_PATH)}`);
  if (account.emailVerified) redirect(next);

  const api = await serverApi();
  const { data: status, response } = await api.GET('/auth/verification');
  if (!status) {
    throw new Error(`Checking the email address failed with HTTP ${String(response.status)}`);
  }

  return <VerifyEmailForm status={status} next={next} />;
}
