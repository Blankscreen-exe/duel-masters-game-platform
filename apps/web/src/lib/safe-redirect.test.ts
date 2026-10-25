import { describe, expect, it } from 'vitest';
import { safeRedirect } from './safe-redirect';

describe('safeRedirect', () => {
  it('keeps pages of this site', () => {
    expect(safeRedirect('/cards')).toBe('/cards');
    expect(safeRedirect('/cards?q=bolshack#top')).toBe('/cards?q=bolshack#top');
    expect(safeRedirect(['/decks', '/cards'])).toBe('/decks');
  });

  it('goes home without a page', () => {
    expect(safeRedirect(undefined)).toBe('/');
    expect(safeRedirect('')).toBe('/');
    expect(safeRedirect([])).toBe('/');
  });

  it('never leads to another site', () => {
    for (const next of [
      'https://evil.example',
      '//evil.example',
      '/\\evil.example',
      '\\\\evil.example',
      'javascript:alert(1)',
      'cards',
    ]) {
      expect(safeRedirect(next), next).toBe('/');
    }
  });
});
