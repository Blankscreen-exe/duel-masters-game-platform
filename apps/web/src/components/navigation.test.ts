import { describe, expect, it } from 'vitest';
import { isCurrent } from './navigation';

describe('isCurrent', () => {
  const cards = { href: '/cards', label: 'Cards' };

  it('matches the page and the pages below it', () => {
    expect(isCurrent(cards, '/cards')).toBe(true);
    expect(isCurrent(cards, '/cards/0079')).toBe(true);
  });

  it('does not match other pages, even with the same start', () => {
    expect(isCurrent(cards, '/')).toBe(false);
    expect(isCurrent(cards, '/cardsets')).toBe(false);
  });

  it('matches home only on the home page', () => {
    const home = { href: '/', label: 'Home' };

    expect(isCurrent(home, '/')).toBe(true);
    expect(isCurrent(home, '/cards')).toBe(false);
  });
});
