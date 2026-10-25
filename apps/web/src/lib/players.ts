import type { components } from './api/schema';
import { firstParam, hrefWithQuery, wholeNumberParam, type SearchParams } from './search-params';

type Schemas = components['schemas'];
export type Rank = Schemas['Rank'];
export type Tier = Schemas['Tier'];
export type RankedPlayer = Schemas['RankedPlayer'];
export type Rankings = Schemas['Rankings'];
export type PlayerProfile = Schemas['PlayerProfile'];
export type Match = Schemas['Match'];
export type MatchPlayer = Schemas['MatchPlayer'];
export type MatchPage = Schemas['MatchPage'];

/** The longest name search the rankings take: a name's longest. */
export const RANKINGS_SEARCH_MAX_LENGTH = 20;

/** A player's profile page. */
export function profileHref(username: string): string {
  return `/u/${encodeURIComponent(username)}`;
}

/** A page of a player's match history. */
export function matchesHref(username: string, page = 1): string {
  return hrefWithQuery(`${profileHref(username)}/matches`, { page });
}

/** The rankings as a page address asks for them. */
export interface RankingsQuery {
  readonly search?: string;
  readonly page?: number;
}

/**
 * The rankings asked for in a page address (/rankings?search=…&page=2). Anything not valid is left
 * out, so a hand-edited address shows the full rankings rather than an error.
 */
export function parseRankingsQuery(params: SearchParams): RankingsQuery {
  const search = firstParam(params['search'])?.trim().slice(0, RANKINGS_SEARCH_MAX_LENGTH);
  const page = wholeNumberParam(params['page'], 1);

  return { ...(search ? { search } : {}), ...(page === undefined ? {} : { page }) };
}

/** A page of the rankings. */
export function rankingsHref(query: RankingsQuery, page = query.page ?? 1): string {
  return hrefWithQuery('/rankings', { search: query.search, page });
}

/** The page of the full rankings a place is on. */
export function pageOfPlace(place: number, pageSize: number): number {
  return Math.max(1, Math.ceil(place / pageSize));
}

/** A streak in short, such as W3 for three wins in a row, or L2; a dash for none. */
export function streakText(streak: number): string {
  if (streak === 0) return '—';
  return streak > 0 ? `W${String(streak)}` : `L${String(-streak)}`;
}

/** A streak in words, for screen readers and tooltips. */
export function streakDescription(streak: number): string {
  if (streak === 0) return 'No streak';
  const games = Math.abs(streak);
  const kind = streak > 0 ? 'win' : 'loss';
  return games === 1
    ? `1 ${kind}`
    : `${String(games)} ${kind === 'win' ? 'wins' : 'losses'} in a row`;
}

/** A rating change with its sign, such as +16 or −16. */
export function ratingChangeText(change: number): string {
  if (change > 0) return `+${String(change)}`;
  if (change < 0) return `−${String(-change)}`;
  return '0';
}

/** The share of games won, such as 67%; a dash before any game counted. */
export function winRate(wins: number, losses: number): string {
  const played = wins + losses;
  return played === 0 ? '—' : `${String(Math.round((wins / played) * 100))}%`;
}

/** A match as one of its players saw it: their side, their opponent's, and how it went. */
export interface MatchSide {
  readonly player: MatchPlayer;
  readonly opponent: MatchPlayer;
  readonly result: 'won' | 'lost' | 'none';
}

/** A match from a player's side; from the host's for anyone else. */
export function matchSide(match: Match, playerId: string): MatchSide {
  const [player, opponent] =
    match.guest.id === playerId ? [match.guest, match.host] : [match.host, match.guest];

  let result: MatchSide['result'] = 'none';
  if (match.winnerId !== undefined) result = match.winnerId === player.id ? 'won' : 'lost';

  return { player, opponent, result };
}

/** How a match ended, in a word or two, for lists of matches. */
export const MATCH_ENDING: Readonly<Record<Match['ending'], string>> = {
  play: 'Played out',
  surrender: 'Surrender',
  disconnect: 'Left the game',
  aborted: 'Stopped',
};

/** The words for a match's result, from a player's side. */
export const MATCH_RESULT: Readonly<Record<MatchSide['result'], string>> = {
  won: 'Won',
  lost: 'Lost',
  none: 'No result',
};

const DATE = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  timeZone: 'UTC',
});

/** The day a match was played, such as 25 Oct 2026, the same on the server and in the browser. */
export function matchDate(endedAt: string): string {
  return DATE.format(new Date(endedAt));
}
