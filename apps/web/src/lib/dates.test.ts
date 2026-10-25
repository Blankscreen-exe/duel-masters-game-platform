import { describe, expect, it } from 'vitest';
import { dateTimeText, dayText } from './dates';

describe('dates', () => {
  it('names the day in UTC', () => {
    expect(dayText('2026-10-25T23:30:00Z')).toBe('25 Oct 2026');
  });

  it('names the moment in UTC', () => {
    expect(dateTimeText('2026-11-02T12:05:00Z')).toBe('2 Nov 2026, 12:05 UTC');
  });
});
