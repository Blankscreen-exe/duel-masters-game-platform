import { describe, expect, it } from 'vitest';
import { splitReminder } from './abilities';

describe('splitReminder', () => {
  it('takes the reminder off the end of the line', () => {
    expect(
      splitReminder({
        text: 'Double breaker (This creature breaks 2 shields.)',
        reminder: '(This creature breaks 2 shields.)',
      }),
    ).toEqual({ rule: 'Double breaker', reminder: '(This creature breaks 2 shields.)' });
  });

  it('keeps a line without a reminder whole', () => {
    const text = 'At the end of each of your turns, you may untap this creature.';
    expect(splitReminder({ text })).toEqual({ rule: text });
  });

  it('keeps the line whole when the reminder is not at its end, or is all of it', () => {
    expect(splitReminder({ text: '(Note) Blocker', reminder: '(Note)' })).toEqual({
      rule: '(Note) Blocker',
    });
    expect(splitReminder({ text: '(Note)', reminder: '(Note)' })).toEqual({ rule: '(Note)' });
  });
});
