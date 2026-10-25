import type { Metadata } from 'next';
import Link from 'next/link';
import { notFound } from 'next/navigation';
import { cache } from 'react';
import { CardImage } from '@/components/cards/card-image';
import { Panel } from '@/components/ui/panel';
import type { Card } from '@/lib/api/client';
import { serverApi } from '@/lib/api/server';
import { splitReminder } from '@/lib/abilities';
import { civilizationKey } from '@/lib/civilizations';

/** The card, asked once per request for both the page and its title. */
const getCard = cache(async (cardId: string): Promise<Card> => {
  const api = await serverApi();
  const { data, response } = await api.GET('/cards/{cardId}', { params: { path: { cardId } } });

  if (data) return data;
  if (response.status === 404) notFound();

  throw new Error(`Loading the card failed with HTTP ${String(response.status)}`);
});

export async function generateMetadata({
  params,
}: PageProps<'/cards/[cardId]'>): Promise<Metadata> {
  const card = await getCard((await params).cardId);
  return { title: card.name };
}

export default async function CardPage({ params }: PageProps<'/cards/[cardId]'>) {
  const card = await getCard((await params).cardId);
  const accent = card.civilizations.map(civilizationKey).find((key) => key !== undefined);

  return (
    <div className="space-y-4">
      <Link
        href="/cards"
        className="font-display text-sm font-semibold tracking-wider text-slate-400 uppercase hover:text-white"
      >
        ← All cards
      </Link>

      <div className="grid gap-6 md:grid-cols-[minmax(0,22rem)_1fr]">
        <CardImage
          src={card.imageUrl}
          name={card.name}
          priority
          className="mx-auto w-full max-w-xs md:max-w-none"
        />

        <div className="space-y-6">
          <Panel accent={accent} className="p-5">
            <h1 className="font-display text-3xl font-bold tracking-wide text-white sm:text-4xl">
              {card.name}
            </h1>
            <dl className="mt-4 grid grid-cols-2 gap-x-6 gap-y-3 text-sm sm:grid-cols-4">
              <Fact label="Civilization" value={card.civilizations.join(' / ')} />
              <Fact label="Type" value={card.type} />
              <Fact label="Cost" value={String(card.cost)} />
              {card.powerText !== '' && <Fact label="Power" value={card.powerText} />}
              {card.races.length > 0 && (
                <Fact
                  label={card.races.length === 1 ? 'Race' : 'Races'}
                  value={card.races.join(' / ')}
                />
              )}
            </dl>

            {card.abilities.length > 0 && (
              <ul className="mt-5 space-y-2 border-t border-slate-700/60 pt-4 text-slate-200">
                {card.abilities.map((ability, index) => {
                  const { rule, reminder } = splitReminder(ability);
                  return (
                    // Abilities have no id of their own and never change order.
                    <li key={index}>
                      {rule}
                      {reminder && <span className="text-slate-400 italic"> {reminder}</span>}
                    </li>
                  );
                })}
              </ul>
            )}

            {!card.playable && (
              <p className="mt-5 bg-slate-800/60 px-3 py-2 text-sm text-slate-400 ring-1 ring-slate-600/50 ring-inset">
                The rules engine cannot play this card yet, so it cannot go in decks.
              </p>
            )}
          </Panel>

          <Panel title={card.printings.length === 1 ? 'Printing' : 'Printings'}>
            <table className="w-full text-left text-sm">
              <thead className="text-[11px] tracking-wider text-slate-500 uppercase">
                <tr className="border-b border-slate-700/60">
                  <th scope="col" className="px-4 py-2 font-semibold">
                    Set
                  </th>
                  <th scope="col" className="px-4 py-2 font-semibold">
                    Number
                  </th>
                  <th scope="col" className="px-4 py-2 font-semibold">
                    Rarity
                  </th>
                </tr>
              </thead>
              <tbody>
                {card.printings.map((printing) => (
                  <tr
                    key={printing.id}
                    className="border-b border-slate-800/80 last:border-0 hover:bg-white/[0.03]"
                  >
                    <td className="px-4 py-2 font-semibold text-slate-200">{printing.set}</td>
                    <td className="px-4 py-2 text-slate-300">{printing.collectorNumber}</td>
                    <td className="px-4 py-2 text-slate-300">{printing.rarity}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Panel>
        </div>
      </div>
    </div>
  );
}

function Fact({ label, value }: { readonly label: string; readonly value: string }) {
  return (
    <div>
      <dt className="text-[11px] font-semibold tracking-wider text-slate-500 uppercase">{label}</dt>
      <dd className="mt-0.5 font-semibold text-slate-100">{value}</dd>
    </div>
  );
}
