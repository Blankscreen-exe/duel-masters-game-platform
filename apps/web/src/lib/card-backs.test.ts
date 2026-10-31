import { existsSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import { CARD_BACK, CARD_BACKS } from './card-backs';

describe('card backs', () => {
  it.each(Object.entries(CARD_BACKS))('has the %s picture, ready to switch to', (_name, path) => {
    expect(existsSync(fileURLToPath(new URL(`../../public${path}`, import.meta.url)))).toBe(true);
  });

  it('uses one of them', () => {
    expect(Object.values(CARD_BACKS)).toContain(CARD_BACK);
  });
});
