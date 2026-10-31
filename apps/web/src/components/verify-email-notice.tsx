import { getSessionAccount } from '@/lib/session';
import { VerifyEmailStrip } from './verify-email-strip';

/** Reminds a player who has not confirmed their email address, on the pages they can still see. */
export async function VerifyEmailNotice() {
  const account = await getSessionAccount();
  if (!account || account.emailVerified) return null;

  return <VerifyEmailStrip />;
}
