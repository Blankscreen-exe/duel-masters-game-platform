import type { Metadata } from 'next';
import Link from 'next/link';
import { FormatRowActions } from '@/components/admin/format-row-actions';
import { buttonClasses } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import { serverApi } from '@/lib/api/server';
import { formatRulesText } from '@/lib/formats';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Formats · Admin' };

/** Every format, in the order players see them, to change, order, retire or bring back. */
export default async function AdminFormatsPage() {
  await requireAdmin('/admin/formats');
  const api = await serverApi();
  const { data: formats } = await api.GET('/admin/formats');

  if (!formats) throw new Error('The formats failed to load');

  const order = formats.map(({ id }) => id);

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          Formats
        </h1>
        <Link href="/admin/formats/new" className={buttonClasses('primary')}>
          Add a format
        </Link>
      </header>
      <p className="text-sm text-slate-400">
        Players see formats in this order, and the first they can choose is picked for them to begin
        with.
      </p>

      <Panel>
        <ul aria-label={`Formats, ${String(formats.length)} in all`}>
          {formats.map((format) => (
            <li
              key={format.id}
              className="flex flex-wrap items-center justify-between gap-x-6 gap-y-3 border-b border-slate-800/80 px-4 py-3 last:border-0 hover:bg-white/[0.03]"
            >
              <div className="min-w-0 flex-1 basis-72 space-y-1">
                <p className="flex flex-wrap items-center gap-2">
                  <Link
                    href={`/admin/formats/${encodeURIComponent(format.id)}`}
                    className="font-semibold text-slate-100 hover:text-accent"
                  >
                    {format.name}
                  </Link>
                  {format.retired && (
                    <span className="px-1.5 py-0.5 text-[10px] font-bold tracking-wider text-slate-400 uppercase ring-1 ring-slate-500/50 ring-inset">
                      Retired
                    </span>
                  )}
                </p>
                <p className="text-sm text-slate-300">{format.description}</p>
                <p className="text-xs text-slate-500">
                  {formatRulesText(format)}
                  {format.opponents.length > 0 &&
                    ` · Computer opponents: ${format.opponents.join(', ')}`}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Link
                  href={`/admin/formats/${encodeURIComponent(format.id)}`}
                  className={buttonClasses('quiet')}
                >
                  Edit
                </Link>
                <FormatRowActions format={format} order={order} />
              </div>
            </li>
          ))}
        </ul>
      </Panel>
    </div>
  );
}
