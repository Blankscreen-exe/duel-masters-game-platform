import { describe, expect, it } from 'vitest';
import {
  BAN_LENGTHS,
  bannedMessage,
  inForce,
  MUTE_LENGTHS,
  mutedMessage,
  penaltyEnd,
} from './moderation';

describe('lengths', () => {
  it('offers timed bans and mutes, and one for good', () => {
    expect(BAN_LENGTHS.map((length) => length.minutes)).toEqual([1440, 10080, 43200, undefined]);
    expect(MUTE_LENGTHS.map((length) => length.minutes)).toEqual([60, 1440, 10080, undefined]);
  });
});

describe('penaltyEnd', () => {
  it('names the moment in UTC, or says for good', () => {
    expect(penaltyEnd('2026-11-02T12:05:00Z')).toBe('until 2 Nov 2026, 12:05 UTC');
    expect(penaltyEnd(undefined)).toBe('for good');
  });
});

describe('inForce', () => {
  const now = Date.parse('2026-11-01T12:00:00Z');

  it('is on until it ends, and for good always', () => {
    expect(inForce(undefined, now)).toBe(false);
    expect(inForce({ since: '2026-11-01T00:00:00Z' }, now)).toBe(true);
    expect(inForce({ since: '2026-11-01T00:00:00Z', until: '2026-11-01T12:00:01Z' }, now)).toBe(
      true,
    );
    expect(inForce({ since: '2026-11-01T00:00:00Z', until: '2026-11-01T12:00:00Z' }, now)).toBe(
      false,
    );
  });
});

describe('messages', () => {
  it('tells a banned player why and until when', () => {
    expect(
      bannedMessage({
        code: 'banned',
        message: 'This account is banned.',
        until: '2026-11-08T09:30:00Z',
        reason: 'Spamming the chat',
      }),
    ).toBe('This account is banned until 8 Nov 2026, 09:30 UTC. Reason: Spamming the chat');
    expect(bannedMessage({ code: 'banned', message: '', reason: 'Cheating' })).toBe(
      'This account is banned for good. Reason: Cheating',
    );
  });

  it('tells a muted player until when', () => {
    expect(mutedMessage({ since: '2026-11-01T00:00:00Z', until: '2026-11-01T13:00:00Z' })).toBe(
      'An admin has muted you in the chat until 1 Nov 2026, 13:00 UTC.',
    );
    expect(mutedMessage({ since: '2026-11-01T00:00:00Z' })).toBe(
      'An admin has muted you in the chat for good.',
    );
  });
});
