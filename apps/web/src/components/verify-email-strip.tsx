'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { VERIFY_EMAIL_PATH } from '@/lib/verification';

/** The reminder itself, left out on the page it leads to. */
export function VerifyEmailStrip() {
  const pathname = usePathname();
  if (pathname === VERIFY_EMAIL_PATH) return null;

  return (
    <p className="bg-amber-500/10 px-4 py-2 text-center text-sm text-amber-100 ring-1 ring-amber-400/30 ring-inset">
      Confirm your email address to start playing.{' '}
      <Link
        href={VERIFY_EMAIL_PATH}
        className="font-semibold text-accent underline-offset-4 hover:underline"
      >
        Enter your code
      </Link>
    </p>
  );
}
