import type { Card } from '@/lib/api/client';
import { splitReminder } from '@/lib/abilities';

interface CardFactsProps {
  readonly card: Card;
  /** How the facts are laid out across the space they have. */
  readonly columns?: string;
}

/** What a card is and what it does, in words. */
export function CardFacts({ card, columns = 'grid-cols-2 sm:grid-cols-4' }: CardFactsProps) {
  return (
    <>
      <dl className={`grid gap-x-6 gap-y-3 text-sm ${columns}`}>
        <Fact label="Civilization" value={card.civilizations.join(' / ')} />
        <Fact label="Type" value={card.type} />
        <Fact label="Cost" value={String(card.cost)} />
        {card.powerText !== '' && <Fact label="Power" value={card.powerText} />}
        {card.races.length > 0 && (
          <Fact label={card.races.length === 1 ? 'Race' : 'Races'} value={card.races.join(' / ')} />
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
    </>
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
