import { civilizationColour } from '@/lib/civilizations';

/** A small diamond of each civilization's colour, named for screen readers. */
export function CivilizationMarks({
  civilizations,
}: {
  readonly civilizations: readonly string[];
}) {
  if (civilizations.length === 0) return null;

  return (
    <span
      role="img"
      aria-label={civilizations.join(', ')}
      className="inline-flex items-center gap-1"
    >
      {civilizations.map((civilization) => (
        <span
          key={civilization}
          title={civilization}
          className={`size-2.5 rotate-45 ring-1 ring-black/40 ${civilizationColour(civilization).background}`}
        />
      ))}
    </span>
  );
}
