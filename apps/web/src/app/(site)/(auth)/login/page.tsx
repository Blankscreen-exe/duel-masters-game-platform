import type { Metadata } from 'next';
import { redirectIfLoggedIn } from '@/lib/session';
import { safeRedirect } from '@/lib/safe-redirect';
import { AuthForm } from '../auth-form';

export const metadata: Metadata = { title: 'Log in' };

export default async function LogInPage({ searchParams }: PageProps<'/login'>) {
  const next = safeRedirect((await searchParams)['next']);

  await redirectIfLoggedIn(next);

  return <AuthForm mode="login" next={next} />;
}
