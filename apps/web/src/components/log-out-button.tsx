'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { api } from '@/lib/api/client';
import { Button } from './ui/button';

/** Ends the session, then shows the page again as a visitor sees it. */
export function LogOutButton() {
  const router = useRouter();
  const [pending, setPending] = useState(false);

  async function logOut() {
    setPending(true);
    try {
      // The server clears the cookie; logging out without a session is not an error.
      await api.POST('/auth/logout', { headers: { 'Content-Type': 'application/json' } });
      router.refresh();
    } finally {
      setPending(false);
    }
  }

  return (
    <Button variant="quiet" disabled={pending} onClick={() => void logOut()}>
      {pending ? 'Logging out…' : 'Log out'}
    </Button>
  );
}
