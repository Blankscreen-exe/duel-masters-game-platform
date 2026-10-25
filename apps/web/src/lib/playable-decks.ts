import 'server-only';
import { serverApi } from './api/server';
import type { PlayableDecks } from './lobby';

/** The decks the logged-in player may play, for the pages where they choose one. */
export async function getPlayableDecks(): Promise<PlayableDecks> {
  const api = await serverApi();
  const [starters, own] = await Promise.all([api.GET('/starter-decks'), api.GET('/decks')]);
  if (!starters.data || !own.data) throw new Error('Your decks could not be loaded');

  return { starters: starters.data, own: own.data };
}
