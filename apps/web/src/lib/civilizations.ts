import type { Civilization } from '@/components/ui/panel';

/** The civilizations in the game's own order, as the card database names them. */
export const CIVILIZATIONS: readonly { readonly name: string; readonly key: Civilization }[] = [
  { name: 'Light', key: 'light' },
  { name: 'Water', key: 'water' },
  { name: 'Darkness', key: 'darkness' },
  { name: 'Fire', key: 'fire' },
  { name: 'Nature', key: 'nature' },
];

/** The theme's key for a civilization named by the card database, if it is one of the five. */
export function civilizationKey(name: string): Civilization | undefined {
  return CIVILIZATIONS.find((civilization) => civilization.name === name)?.key;
}
