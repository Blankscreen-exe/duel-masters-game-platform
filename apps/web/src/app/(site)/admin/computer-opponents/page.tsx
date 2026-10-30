import type { Metadata } from 'next';
import Link from 'next/link';
import { OpponentRowActions } from '@/components/admin/opponent-row-actions';
import { CivilizationMarks } from '@/components/cards/civilization-marks';
import { buttonClasses } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import { serverApi } from '@/lib/api/server';
import { rulesText, type AdminComputerOpponent } from '@/lib/computer-opponents';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Computer opponents · Admin' };

/** Every computer opponent, offered or retired, to change, offer or retire. */
export default async function AdminComputerOpponentsPage() {
  await requireAdmin('/admin/computer-opponents');
  const api = await serverApi();
  const [opponents, options] = await Promise.all([
    api.GET('/admin/computer-opponents'),
    api.GET('/admin/computer-opponents/options'),
  ]);

  if (!opponents.data || !options.data) {
    throw new Error('The computer opponents failed to load');
  }

  const styleLabel = new Map(options.data.styles.map(({ name, label }) => [name, label]));

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          Computer opponents
        </h1>
        <Link href="/admin/computer-opponents/new" className={buttonClasses('primary')}>
          Add an opponent
        </Link>
      </header>

      <Panel>
        {opponents.data.length === 0 ? (
          <p className="px-4 py-8 text-center text-slate-400">No computer opponents yet.</p>
        ) : (
          <ul aria-label={`Computer opponents, ${String(opponents.data.length)} in all`}>
            {opponents.data.map((opponent) => (
              <OpponentRow
                key={opponent.id}
                opponent={opponent}
                style={styleLabel.get(opponent.style) ?? opponent.style}
              />
            ))}
          </ul>
        )}
      </Panel>
    </div>
  );
}

function OpponentRow({
  opponent,
  style,
}: {
  readonly opponent: AdminComputerOpponent;
  readonly style: string;
}) {
  const strength = opponent.strength === 'easy' ? 'Easy' : 'Normal';

  return (
    <li className="flex flex-wrap items-center justify-between gap-x-6 gap-y-3 border-b border-slate-800/80 px-4 py-3 last:border-0 hover:bg-white/[0.03]">
      <div className="min-w-0 flex-1 basis-72 space-y-1">
        <p className="flex flex-wrap items-center gap-2">
          <Link
            href={`/admin/computer-opponents/${encodeURIComponent(opponent.id)}`}
            className="font-semibold text-slate-100 hover:text-accent"
          >
            {opponent.name}
          </Link>
          <CivilizationMarks civilizations={opponent.civilizations} />
          <span
            className={`px-1.5 py-0.5 text-[10px] font-bold tracking-wider uppercase ring-1 ring-inset ${
              opponent.offered
                ? 'text-emerald-300 ring-emerald-400/50'
                : 'text-slate-400 ring-slate-500/50'
            }`}
          >
            {opponent.offered ? 'Offered' : 'Retired'}
          </span>
        </p>
        <p className="text-sm text-slate-300">{opponent.description}</p>
        <p className="text-xs text-slate-500">
          {style} · {strength} · {opponent.formatName} format · {rulesText(opponent)} ·{' '}
          {opponent.cardCount} cards
        </p>
        {opponent.problems.length > 0 && (
          <p className="text-xs text-red-300">
            Cannot play: {opponent.problems[0]?.message}
            {opponent.problems.length > 1 && ` (and ${String(opponent.problems.length - 1)} more)`}
          </p>
        )}
      </div>
      <div className="flex items-center gap-2">
        <Link
          href={`/admin/computer-opponents/${encodeURIComponent(opponent.id)}`}
          className={buttonClasses('quiet')}
        >
          Edit
        </Link>
        <OpponentRowActions opponent={opponent} />
      </div>
    </li>
  );
}
