import { describe, expect, it } from 'vitest';
import { minutesAndSeconds } from './time-text';

describe('minutesAndSeconds', () => {
  it('shows minutes and two-digit seconds', () => {
    expect(minutesAndSeconds(754)).toBe('12:34');
    expect(minutesAndSeconds(5)).toBe('0:05');
    expect(minutesAndSeconds(120)).toBe('2:00');
  });

  it('drops parts of a second and never goes below nothing', () => {
    expect(minutesAndSeconds(59.9)).toBe('0:59');
    expect(minutesAndSeconds(-3)).toBe('0:00');
  });
});
