'use client';

import { useRouter } from 'next/navigation';
import type { AdminComputerOpponent } from '@/lib/computer-opponents';
import { OpponentOfferButton } from './opponent-offer-button';

/** The list's way to offer or retire an opponent, which draws the list again afterwards. */
export function OpponentRowActions({
  opponent,
}: {
  readonly opponent: Pick<AdminComputerOpponent, 'id' | 'name' | 'offered'>;
}) {
  const router = useRouter();

  return (
    <OpponentOfferButton
      opponent={opponent}
      onChanged={() => {
        router.refresh();
      }}
    />
  );
}
