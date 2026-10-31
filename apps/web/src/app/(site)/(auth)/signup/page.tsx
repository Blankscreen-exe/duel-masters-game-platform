import type { Metadata } from 'next';
import { redirectIfLoggedIn } from '@/lib/session';
import { safeRedirect } from '@/lib/safe-redirect';
import { AuthForm } from '../auth-form';

export const metadata: Metadata = { title: 'Sign up' };

export default async function SignUpPage({ searchParams }: PageProps<'/signup'>) {
  const next = safeRedirect((await searchParams)['next']);

  await redirectIfLoggedIn(next);

  return <AuthForm mode="signup" next={next} />;
}
