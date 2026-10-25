import { describe, expect, it } from 'vitest';
import { civilizationColour, civilizationKey } from './civilizations';

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

describe('civilizationColour', () => {
  it('colours a civilization given by key or by name', () => {
    expect(civilizationColour('fire')).toEqual({
      background: 'bg-civ-fire',
      border: 'border-civ-fire',
    });
    expect(civilizationColour('Water').background).toBe('bg-civ-water');
  });

  it('greys out anything else', () => {
    expect(civilizationColour('Zero').background).toBe('bg-slate-500');
  });
});
