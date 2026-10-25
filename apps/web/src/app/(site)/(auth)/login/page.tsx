import type { Metadata } from 'next';
import { redirect } from 'next/navigation';
import { getCurrentAccount } from '@/lib/session';
import { safeRedirect } from '@/lib/safe-redirect';
import { AuthForm } from '../auth-form';

export const metadata: Metadata = { title: 'Log in' };

export default async function LogInPage({ searchParams }: PageProps<'/login'>) {
  const next = safeRedirect((await searchParams)['next']);

  // Already logged in: nothing to do here.
  if (await getCurrentAccount()) redirect(next);

  return <AuthForm mode="login" next={next} />;
}
