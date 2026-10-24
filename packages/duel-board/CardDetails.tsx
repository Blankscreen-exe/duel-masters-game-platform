import { attackBonus, describePower } from "./creaturePower";
import type { StatusBadge } from "./creatureStatus";
import { StatusList } from "./StatusBadges";
import type { CreaturePower } from "./types";

/** What the board knows about a creature in play beyond its printed card: its
 * power and its statuses. Shown when the card is viewed. */
export function CardDetails({
  power,
  statuses,
}: {
  power?: CreaturePower | null;
  statuses: StatusBadge[];
}) {
  if (!power && statuses.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg bg-gray-900/90 px-3 py-2 text-white">
      {power && <PowerLine power={power} />}
      <StatusList badges={statuses} />
    </div>
  );
}

/** The creature's power, with the printed power crossed out when effects
 * have changed it. */
function PowerLine({ power }: { power: CreaturePower }) {
  const changed = power.current !== power.printed;
  const bonus = attackBonus(power);

  return (
    <p
      aria-label={describePower(power)}
      className="text-sm font-semibold leading-tight"
    >
      <span className="text-gray-300">Power </span>
      {changed && <s className="mr-1 text-gray-400">{power.printed}</s>}
      <span
        className={
          power.current > power.printed
            ? "text-green-400"
            : power.current < power.printed
              ? "text-red-400"
              : ""
        }
      >
        {power.current}
      </span>
      {bonus > 0 && (
        <span className="ml-2 whitespace-nowrap text-amber-400">
          ⚔ {power.attacking} attacking
        </span>
      )}
    </p>
  );
}
