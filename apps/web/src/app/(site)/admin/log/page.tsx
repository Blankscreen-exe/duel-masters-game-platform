import type { Metadata } from 'next';
import { Button } from '@/components/ui/button';
import { Pagination } from '@/components/ui/pagination';
import { Panel } from '@/components/ui/panel';
import { SelectField } from '@/components/ui/select-field';
import {
  ADMIN_LOG_ACTIONS,
  adminLogHref,
  parseAdminLogQuery,
  type AdminLogAction,
  type AdminLogEntry,
} from '@/lib/admin';
import { serverApi } from '@/lib/api/server';
import { dateTimeText } from '@/lib/dates';
import { pageCount } from '@/lib/search-params';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Admin log · Admin' };

/** The columns of the log, as a grid on wider screens; on phones each entry stacks. */
const COLUMNS = 'md:grid md:grid-cols-[11rem_8rem_11rem_minmax(0,1fr)] md:gap-3';

/** What admins have done in the admin panel, the latest first, of one action or all. */
export default async function AdminLogPage({ searchParams }: PageProps<'/admin/log'>) {
  const query = parseAdminLogQuery(await searchParams);
  await requireAdmin(adminLogHref(query));
  const api = await serverApi();
  const { data: log, response } = await api.GET('/admin/log', { params: { query } });

  if (!log) {
    throw new Error(`The admin log failed to load with HTTP ${String(response.status)}`);
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          Admin log
        </h1>
        <form action="/admin/log" className="flex items-end gap-2">
          <SelectField
            label="Show"
            name="action"
            defaultValue={query.action ?? ''}
            className="w-60"
          >
            <option value="">Everything</option>
            {(Object.keys(ADMIN_LOG_ACTIONS) as AdminLogAction[]).map((action) => (
              <option key={action} value={action}>
                {ADMIN_LOG_ACTIONS[action]}
              </option>
            ))}
          </SelectField>
          <Button type="submit" variant="secondary">
            Show
          </Button>
        </form>
      </header>

      <Panel>
        {log.entries.length === 0 ? (
          <p className="px-4 py-8 text-center text-slate-400">
            {query.action
              ? 'Nothing of this kind yet.'
              : 'Nothing has been done in the admin panel yet.'}
          </p>
        ) : (
          <>
            <div
              aria-hidden
              className={`hidden border-b border-slate-700/60 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-slate-400 uppercase ${COLUMNS}`}
            >
              <span>When</span>
              <span>Admin</span>
              <span>Did</span>
              <span>To</span>
            </div>
            <ol aria-label="Admin log, the latest first">
              {log.entries.map((entry) => (
                <LogRow key={entry.id} entry={entry} />
              ))}
            </ol>
          </>
        )}
      </Panel>

      <Pagination
        page={log.page}
        pages={pageCount(log.total, log.pageSize)}
        href={(number) => adminLogHref(query, number)}
      />
    </div>
  );
}

function LogRow({ entry }: { readonly entry: AdminLogEntry }) {
  return (
    <li
      className={`flex flex-wrap gap-x-3 gap-y-1 border-b border-slate-800/80 px-4 py-3 text-sm last:border-0 hover:bg-white/[0.03] ${COLUMNS}`}
    >
      <time
        dateTime={entry.at}
        className="basis-full text-xs text-slate-400 md:basis-auto md:text-sm"
      >
        {dateTimeText(entry.at)}
      </time>
      <span className="font-semibold text-slate-200">{entry.adminName}</span>
      <span className="text-accent">{ADMIN_LOG_ACTIONS[entry.action]}</span>
      <span className="min-w-0 basis-full break-words text-slate-200 md:basis-auto">
        {entry.subject}
        {entry.details && <span className="block text-xs text-slate-400">{entry.details}</span>}
      </span>
    </li>
  );
}
