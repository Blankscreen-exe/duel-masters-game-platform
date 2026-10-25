/** One of the game's five civilizations, as the theme names its colours. */
export type Civilization = 'light' | 'water' | 'darkness' | 'fire' | 'nature';

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

/**
 * Each civilization's colour as style classes, written out in full so the stylesheet includes
 * them. A name that is not one of the five gets grey.
 */
const COLOUR_CLASSES: Record<
  Civilization,
  { readonly background: string; readonly border: string }
> = {
  light: { background: 'bg-civ-light', border: 'border-civ-light' },
  water: { background: 'bg-civ-water', border: 'border-civ-water' },
  darkness: { background: 'bg-civ-darkness', border: 'border-civ-darkness' },
  fire: { background: 'bg-civ-fire', border: 'border-civ-fire' },
  nature: { background: 'bg-civ-nature', border: 'border-civ-nature' },
};

const UNKNOWN_COLOUR = { background: 'bg-slate-500', border: 'border-slate-500' } as const;

/**
 * The style classes colouring something in a civilization's colour, given its key ("fire") or
 * its name in the card database ("Fire").
 */
export function civilizationColour(civilization: string): {
  readonly background: string;
  readonly border: string;
} {
  const key = CIVILIZATIONS.find((c) => c.key === civilization || c.name === civilization)?.key;
  return key ? COLOUR_CLASSES[key] : UNKNOWN_COLOUR;
}
