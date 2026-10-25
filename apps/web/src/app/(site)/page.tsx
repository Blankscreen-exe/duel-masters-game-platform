import Link from 'next/link';
import { CardImage } from '@/components/cards/card-image';
import { LobbyChatPanel } from '@/components/chat/lobby-chat';
import { SiteStatsLine } from '@/components/live/site-stats';
import { buttonClasses } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import type { Card } from '@/lib/api/client';
import { serverApi } from '@/lib/api/server';
import { getCurrentAccount } from '@/lib/session';

/** One classic card of each civilization: Light, Water, Darkness, Fire and Nature. */
const SHOWCASE_CARD_IDS = ['0001', '0033', '0005', '0079', '0100'];

export default async function HomePage() {
  const api = await serverApi();
  const [account, showcase, stats, chat] = await Promise.all([
    getCurrentAccount(),
    showcaseCards(),
    api.GET('/stats'),
    api.GET('/chat'),
  ]);

  return (
    <div className="space-y-10">
      <section className="cut-corners relative overflow-hidden bg-panel/80 px-6 py-12 ring-1 ring-slate-500/40 ring-inset sm:px-10 sm:py-16">
        <div
          aria-hidden
          className="absolute inset-0 bg-[radial-gradient(ellipse_at_top_right,rgb(56_189_248/0.15),transparent_60%)]"
        />
        <div className="relative max-w-2xl">
          <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase sm:text-5xl">
            Duel Masters, online
          </h1>
          <p className="mt-4 text-lg text-slate-300">
            Build decks from every card of the original game and duel other players, with every rule
            and card effect played out for you.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            {account ? (
              <>
                <p className="w-full font-display text-xl font-semibold tracking-wide text-slate-200">
                  Welcome back, {account.username}.
                </p>
                <Link href="/play" className={buttonClasses('primary', 'text-lg')}>
                  Play now
                </Link>
              </>
            ) : (
              <>
                <Link href="/signup" className={buttonClasses('primary', 'text-lg')}>
                  Sign up to play
                </Link>
                <Link href="/login" className={buttonClasses('secondary', 'text-lg')}>
                  Log in
                </Link>
              </>
            )}
          </div>
          {stats.data && (
            <div className="mt-8">
              <SiteStatsLine initial={stats.data} />
            </div>
          )}
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-[1fr_24rem]">
        {showcase.length > 0 && (
          <Panel title="The five civilizations" accent="fire" className="self-start">
            <ul className="grid grid-cols-3 gap-3 p-4 sm:grid-cols-5 sm:gap-4">
              {showcase.map((card) => (
                <li key={card.id} className="transition hover:-translate-y-1 hover:scale-[1.03]">
                  <CardImage src={card.imageUrl} name={card.name} priority />
                </li>
              ))}
            </ul>
          </Panel>
        )}
        {chat.data && (
          <LobbyChatPanel
            initial={chat.data}
            canPost={account !== null}
            mute={account?.mute}
            className="max-h-[28rem]"
          />
        )}
      </div>
    </div>
  );
}

/** The showcase cards the catalogue has; a missing one is left out rather than failing the page. */
async function showcaseCards(): Promise<Card[]> {
  const api = await serverApi();
  const found = await Promise.all(
    SHOWCASE_CARD_IDS.map(async (cardId) => {
      const { data } = await api.GET('/cards/{cardId}', { params: { path: { cardId } } });
      return data;
    }),
  );

  return found.filter((card) => card !== undefined);
}
