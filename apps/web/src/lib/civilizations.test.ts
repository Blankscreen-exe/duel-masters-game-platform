import { describe, expect, it } from 'vitest';
import { civilizationKey } from './civilizations';

describe('civilizationKey', () => {
  it('knows the five civilizations', () => {
    expect(['Light', 'Water', 'Darkness', 'Fire', 'Nature'].map(civilizationKey)).toEqual([
      'light',
      'water',
      'darkness',
      'fire',
      'nature',
    ]);
  });

  it('has nothing for other names', () => {
    expect(civilizationKey('fire')).toBeUndefined();
    expect(civilizationKey('Zero')).toBeUndefined();
  });
});
