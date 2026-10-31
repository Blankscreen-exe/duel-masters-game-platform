import type { Metadata } from 'next';
import Link from 'next/link';
import { UserManager } from '@/components/admin/user-manager';
import { RankBadge } from '@/components/players/rank-badge';
import { Button } from '@/components/ui/button';
import { Pagination } from '@/components/ui/pagination';
import { Panel } from '@/components/ui/panel';
import { TextField } from '@/components/ui/text-field';
import {
  ADMIN_SEARCH_MAX_LENGTH,
  ADMIN_USER_STATUSES,
  adminUsersHref,
  parseAdminUsersQuery,
  type AdminUser,
} from '@/lib/admin';
import { serverApi } from '@/lib/api/server';
import { dayText } from '@/lib/dates';
import { penaltyEnd } from '@/lib/moderation';
import { profileHref } from '@/lib/players';
import { pageCount } from '@/lib/search-params';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Players · Admin' };

/** The columns of the list, as a grid on wider screens; on phones each player stacks. */
const COLUMNS = 'md:grid md:grid-cols-[minmax(0,2fr)_8rem_5rem_7rem_minmax(0,2fr)_6rem] md:gap-3';

/** Every account, by name, to find a player and ban, mute, change the role of or help them. */
export default async function AdminUsersPage({ searchParams }: PageProps<'/admin/users'>) {
  const query = parseAdminUsersQuery(await searchParams);
  const admin = await requireAdmin(adminUsersHref(query));
  const api = await serverApi();
  const { data: page, response } = await api.GET('/admin/users', { params: { query } });

  if (!page) {
    throw new Error(`The players failed to load with HTTP ${String(response.status)}`);
  }

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-end justify-between gap-4">
        <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
          Players
        </h1>
        <form action="/admin/users" role="search" className="flex items-end gap-2">
          {query.status && <input type="hidden" name="status" value={query.status} />}
          <TextField
            label="Find a player"
            name="search"
            type="search"
            defaultValue={query.search}
            maxLength={ADMIN_SEARCH_MAX_LENGTH}
            className="w-56"
          />
          <Button type="submit" variant="secondary">
            Search
          </Button>
        </form>
      </header>

      <nav aria-label="Show" className="flex flex-wrap gap-2">
        {ADMIN_USER_STATUSES.map(({ status, label }) => {
          const current = query.status === status;
          return (
            <Link
              key={label}
              href={adminUsersHref({ search: query.search, status }, 1)}
              aria-current={current ? 'page' : undefined}
              className={`cut-corners-sm px-3 py-1.5 font-display text-sm font-semibold tracking-wider uppercase ring-1 transition ring-inset ${
                current
                  ? 'bg-slate-200 text-slate-900 ring-slate-200'
                  : 'text-slate-300 ring-slate-600/60 hover:bg-white/10 hover:text-white'
              }`}
            >
              {label}
            </Link>
          );
        })}
      </nav>

      <Panel>
        {page.users.length === 0 ? (
          <p className="px-4 py-8 text-center text-slate-400">
            {query.search ? `No player's name contains “${query.search}”.` : 'Nobody here.'}
          </p>
        ) : (
          <>
            <div
              aria-hidden
              className={`hidden border-b border-slate-700/60 px-4 py-2.5 text-[11px] font-semibold tracking-wider text-slate-400 uppercase ${COLUMNS}`}
            >
              <span>Player</span>
              <span>Rank</span>
              <span className="text-right">Won–lost</span>
              <span>Joined</span>
              <span>Ban or mute</span>
              <span />
            </div>
            <ul aria-label={`Players, ${String(page.total)} in all`}>
              {page.users.map((user) => (
                <UserRow key={user.id} user={user} isYou={user.id === admin.id} />
              ))}
            </ul>
          </>
        )}
      </Panel>

      <Pagination
        page={page.page}
        pages={pageCount(page.total, page.pageSize)}
        href={(number) => adminUsersHref(query, number)}
      />
    </div>
  );
}

function UserRow({ user, isYou }: { readonly user: AdminUser; readonly isYou: boolean }) {
  return (
    <li
      className={`flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-slate-800/80 px-4 py-3 text-sm last:border-0 hover:bg-white/[0.03] md:items-center ${COLUMNS}`}
    >
      <div className="flex min-w-0 basis-full items-center gap-2 md:basis-auto">
        <span
          title={user.online ? 'Online' : 'Offline'}
          className={`size-2 shrink-0 rounded-full ${user.online ? 'bg-emerald-400 shadow-[0_0_6px_rgb(52_211_153/0.8)]' : 'bg-slate-600'}`}
        />
        <span className="sr-only">{user.online ? 'Online' : 'Offline'}</span>
        {user.computer ? (
          <span className="truncate font-semibold text-slate-100">{user.username}</span>
        ) : (
          <Link
            href={profileHref(user.username)}
            className="truncate font-semibold text-slate-100 hover:text-accent"
          >
            {user.username}
          </Link>
        )}
        {user.role === 'admin' && (
          <span className="px-1.5 py-0.5 text-[10px] font-bold tracking-wider text-accent uppercase ring-1 ring-accent/50 ring-inset">
            Admin
          </span>
        )}
        {user.computer && (
          <span className="px-1.5 py-0.5 text-[10px] font-bold tracking-wider text-sky-300 uppercase ring-1 ring-sky-400/50 ring-inset">
            Computer
          </span>
        )}
        {!user.emailVerified && (
          <span className="px-1.5 py-0.5 text-[10px] font-bold tracking-wider whitespace-nowrap text-amber-300 uppercase ring-1 ring-amber-400/50 ring-inset">
            Not confirmed
          </span>
        )}
        {isYou && <span className="text-xs text-accent">(you)</span>}
        {user.email && (
          <span title={user.email} className="min-w-0 truncate text-xs text-slate-500">
            {user.email}
          </span>
        )}
      </div>
      <div>
        <RankBadge rank={user.rank} withRating />
      </div>
      <div className="text-slate-300 tabular-nums md:text-right">
        <span className="md:sr-only">Won–lost </span>
        {user.wins}–{user.losses}
      </div>
      <div className="text-slate-400">
        <span className="md:sr-only">Joined </span>
        <time dateTime={user.joinedAt}>{dayText(user.joinedAt)}</time>
      </div>
      <div className="min-w-0 basis-full text-xs md:basis-auto">
        <Penalties user={user} />
      </div>
      <div className="ml-auto md:ml-0 md:text-right">
        {!isYou && !user.computer && <UserManager user={user} />}
      </div>
    </li>
  );
}

/** A player's ban and mute, while they last. */
function Penalties({ user }: { readonly user: AdminUser }) {
  if (!user.ban && !user.mute) return <span className="hidden text-slate-600 md:inline">—</span>;

  return (
    <div className="space-y-0.5">
      {user.ban && (
        <p className="text-red-300">
          <span className="font-semibold">Banned</span> {penaltyEnd(user.ban.until)}
          <span className="block break-words text-slate-400">“{user.ban.reason}”</span>
        </p>
      )}
      {user.mute && (
        <p className="text-amber-300">
          <span className="font-semibold">Muted</span> {penaltyEnd(user.mute.until)}
        </p>
      )}
    </div>
  );
}
