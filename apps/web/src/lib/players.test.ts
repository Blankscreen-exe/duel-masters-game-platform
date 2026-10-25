import { describe, expect, it } from 'vitest';
import {
  matchesHref,
  matchSide,
  pageOfPlace,
  parseRankingsQuery,
  profileHref,
  rankingsHref,
  ratingChangeText,
  streakDescription,
  streakText,
  winRate,
  type Match,
} from './players';

function match(winnerId?: string): Match {
  return {
    id: 'g1',
    name: 'Friendly',
    formatName: 'Open',
    ending: 'play',
    counted: true,
    turns: 9,
    durationSeconds: 600,
    startedAt: '2026-10-25T11:50:00Z',
    endedAt: '2026-10-25T12:00:00Z',
    ...(winnerId === undefined ? {} : { winnerId }),
    host: { id: 'u1', username: 'Shobu', deckName: 'Fire Rush', civilizations: ['Fire'] },
    guest: { id: 'u2', username: 'Kokujo', deckName: 'Dark Tide', civilizations: ['Darkness'] },
  };
}

describe('profile and history addresses', () => {
  it('names the player safely', () => {
    expect(profileHref('Shobu')).toBe('/u/Shobu');
    expect(profileHref('a b')).toBe('/u/a%20b');
    expect(matchesHref('Shobu')).toBe('/u/Shobu/matches');
    expect(matchesHref('Shobu', 3)).toBe('/u/Shobu/matches?page=3');
  });
});

describe('parseRankingsQuery and rankingsHref', () => {
  it('reads a search and a page, and leaves out what is not valid', () => {
    expect(parseRankingsQuery({ search: ' koku ', page: '2' })).toEqual({
      search: 'koku',
      page: 2,
    });
    expect(parseRankingsQuery({ search: '   ', page: '0' })).toEqual({});
    expect(parseRankingsQuery({ search: 'x'.repeat(30) }).search).toHaveLength(20);
  });

  it('writes them back', () => {
    expect(rankingsHref({})).toBe('/rankings');
    expect(rankingsHref({ search: 'koku', page: 2 })).toBe('/rankings?search=koku&page=2');
    expect(rankingsHref({ search: 'koku', page: 2 }, 1)).toBe('/rankings?search=koku');
  });

  it('finds the page a place is on', () => {
    expect(pageOfPlace(1, 25)).toBe(1);
    expect(pageOfPlace(25, 25)).toBe(1);
    expect(pageOfPlace(26, 25)).toBe(2);
  });
});

describe('streaks, rating changes and win rates', () => {
  it('says them briefly', () => {
    expect(streakText(3)).toBe('W3');
    expect(streakText(-2)).toBe('L2');
    expect(streakText(0)).toBe('—');
    expect(ratingChangeText(16)).toBe('+16');
    expect(ratingChangeText(-16)).toBe('−16');
    expect(ratingChangeText(0)).toBe('0');
    expect(winRate(2, 1)).toBe('67%');
    expect(winRate(0, 0)).toBe('—');
  });

  it('says streaks in words', () => {
    expect(streakDescription(3)).toBe('3 wins in a row');
    expect(streakDescription(-1)).toBe('1 loss');
    expect(streakDescription(-4)).toBe('4 losses in a row');
    expect(streakDescription(0)).toBe('No streak');
  });
});

describe('matchSide', () => {
  it('sees a match from either player', () => {
    const won = matchSide(match('u2'), 'u2');
    expect(won.player.username).toBe('Kokujo');
    expect(won.opponent.username).toBe('Shobu');
    expect(won.result).toBe('won');
    expect(matchSide(match('u2'), 'u1').result).toBe('lost');
  });

  it('has no result for a game without a winner, and the host side for anyone else', () => {
    expect(matchSide(match(), 'u1').result).toBe('none');
    expect(matchSide(match('u1'), 'someone').player.username).toBe('Shobu');
  });
});
