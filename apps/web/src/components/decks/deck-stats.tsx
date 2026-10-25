import { civilizationColour } from '@/lib/civilizations';
import { civilizationSplit, manaCurve, totalCards, type DeckDraft } from '@/lib/deck-draft';

/** The deck's mana curve and how its cards split between the civilizations. */
export function DeckStats({ draft }: { readonly draft: DeckDraft }) {
  const curve = manaCurve(draft);
  const highest = Math.max(1, ...curve.map(({ count }) => count));
  const split = civilizationSplit(draft);
  const total = totalCards(draft);

  return (
    <div className="space-y-6 p-4">
      <section aria-label="Mana curve">
        <h3 className="font-display text-xs font-bold tracking-widest text-slate-400 uppercase">
          Mana curve
        </h3>
        <ol className="mt-3 flex h-32 items-end gap-1.5">
          {curve.map(({ label, count }) => (
            <li
              key={label}
              className="flex h-full flex-1 flex-col items-center justify-end gap-1"
              aria-label={`Cost ${label}: ${String(count)} ${count === 1 ? 'card' : 'cards'}`}
            >
              <span className="text-[11px] font-semibold text-slate-300">
                {count > 0 ? count : ''}
              </span>
              <span
                className="w-full bg-linear-to-t from-action to-accent transition-[height] duration-300"
                style={{ height: `${String((count / highest) * 100)}%` }}
              />
              <span className="font-display text-xs font-bold text-slate-500">{label}</span>
            </li>
          ))}
        </ol>
      </section>

      <section aria-label="Civilizations">
        <h3 className="font-display text-xs font-bold tracking-widest text-slate-400 uppercase">
          Civilizations
        </h3>
        {split.length === 0 ? (
          <p className="mt-2 text-sm text-slate-500">No cards yet.</p>
        ) : (
          <ul className="mt-3 space-y-2">
            {split.map(({ civilization, count }) => (
              <li key={civilization} className="text-sm">
                <div className="flex justify-between text-slate-300">
                  <span>{civilization}</span>
                  <span>{count}</span>
                </div>
                <div className="mt-1 h-1.5 bg-slate-800">
                  <div
                    className={`h-full ${civilizationColour(civilization).background}`}
                    style={{ width: `${String(total === 0 ? 0 : (count / total) * 100)}%` }}
                  />
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
