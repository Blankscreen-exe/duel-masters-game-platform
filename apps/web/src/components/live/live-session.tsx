import { getCurrentAccount } from '@/lib/session';
import { LiveAccount } from './live-provider';
import { LiveNotices } from './live-notices';

/**
 * Who the live connection belongs to, and the notices only players get. Depends on the session
 * cookie, so the root layout streams it in rather than making every page wait for it.
 */
export async function LiveSession() {
  const account = await getCurrentAccount();

  return (
    <>
      <LiveAccount accountId={account?.id ?? null} />
      {account && <LiveNotices />}
    </>
  );
}
