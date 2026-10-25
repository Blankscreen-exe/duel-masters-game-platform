import type { DeckLegality as Legality, Format } from '@/lib/api/client';

interface DeckLegalityProps {
  readonly legality: readonly Legality[];
  readonly formats: readonly Format[];
  /** The deck has changes the server has not checked yet. */
  readonly unsaved: boolean;
}

/**
 * Whether the saved deck can be played in each format, and what breaks the rules. The server
 * checks this, so it shows the deck as last saved.
 */
export function DeckLegality({ legality, formats, unsaved }: DeckLegalityProps) {
  return (
    <div className="space-y-3 p-4">
      {unsaved && (
        <p className="text-xs text-amber-300">Save the deck to check your latest changes.</p>
      )}
      <ul className="space-y-3">
        {legality.map((result) => {
          const format = formats.find(({ id }) => id === result.formatId);
          return (
            <li key={result.formatId}>
              <p className="flex items-center justify-between gap-2">
                <span className="font-display font-semibold tracking-wide text-slate-100">
                  {format?.name ?? 'A format'}
                </span>
                <span
                  className={`px-2 py-0.5 text-[11px] font-bold tracking-wide uppercase ring-1 ring-inset ${
                    result.legal
                      ? 'bg-emerald-500/10 text-emerald-300 ring-emerald-400/40'
                      : 'bg-red-500/10 text-red-300 ring-red-400/40'
                  }`}
                >
                  {result.legal ? 'Legal' : 'Not legal'}
                </span>
              </p>
              {result.problems.length > 0 && (
                <ul className="mt-1 list-disc space-y-0.5 pl-5 text-xs text-slate-400">
                  {result.problems.map((problem) => (
                    <li key={`${problem.code}-${problem.cardId ?? ''}`}>{problem.message}</li>
                  ))}
                </ul>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
