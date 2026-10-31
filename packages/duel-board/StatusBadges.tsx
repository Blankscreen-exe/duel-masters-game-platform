import { lowerEdgeShare } from "./creaturePower";
import type { BoardBadges, StatusBadge, StatusTone } from "./creatureStatus";
import { Icon } from "./icons";

/** Space between the badges and the card's upper edge, as a share of the box. */
const EDGE_GAP = 0.02;

const TONE_CLASSES: Record<StatusTone, string> = {
  attack: "bg-red-600",
  defense: "bg-sky-600",
  support: "bg-emerald-600",
  limit: "bg-slate-600",
  sickness: "bg-violet-600",
};

/** Room above a row of creatures for their badges, which sit above each card
 * rather than over it. Kept whether or not any creature has a status, so the
 * cards do not jump as statuses come and go. Sized from the badge, which
 * scales with the screen. */
export const STATUS_LANE_CLASS =
  "pt-[calc(clamp(0.6rem,1.4vh,0.85rem)*1.6_+_0.35rem)]";

/** A status as a round, icon only badge. */
function Badge({ badge }: { badge: StatusBadge }) {
  return (
    <span
      className={`flex h-[1.6em] w-[1.6em] shrink-0 items-center justify-center rounded-full text-white shadow-[0_1px_4px_rgba(0,0,0,0.9)] ${
        TONE_CLASSES[badge.tone]
      } ${
        // A status another card gave stands out from the card's own.
        badge.granted
          ? "ring-2 ring-yellow-300 shadow-[0_0_6px_rgba(253,224,71,0.8)]"
          : "ring-1 ring-white/40"
      }`}
    >
      <Icon name={badge.icon} />
    </span>
  );
}

/** A creature's most important statuses as a row of badges just above its
 * card, in the room STATUS_LANE_CLASS leaves, with a count of any more that do
 * not fit. Kept upright, like its power badge, so they read the same from
 * either side. */
export function StatusBadges({
  badges,
  rotated,
}: {
  badges: BoardBadges;
  /** Whether the card lies on its side, which brings its upper edge down. */
  rotated: boolean;
}) {
  if (badges.shown.length === 0) {
    return null;
  }

  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute left-1/2 flex -translate-x-1/2 gap-0.5 text-[clamp(0.6rem,1.4vh,0.85rem)]"
      style={{
        bottom: `${(1 - lowerEdgeShare(rotated) + EDGE_GAP) * 100}%`,
      }}
    >
      {badges.shown.map((badge) => (
        <Badge key={badge.id} badge={badge} />
      ))}
      {badges.more > 0 && (
        <span className="flex h-[1.6em] items-center rounded-full bg-slate-800 px-1.5 font-bold leading-none text-white shadow-[0_1px_4px_rgba(0,0,0,0.9)] ring-1 ring-white/40">
          +{badges.more}
        </span>
      )}
    </div>
  );
}

/** The statuses spelled out, for viewing a card, where there is room to read
 * them and no hover to ask. */
export function StatusList({ badges }: { badges: StatusBadge[] }) {
  if (badges.length === 0) {
    return null;
  }

  return (
    <ul className="flex shrink-0 flex-col gap-1.5 text-xs">
      {badges.map((badge) => (
        <li key={badge.id} className="flex items-center gap-2">
          <span className="text-sm">
            <Badge badge={badge} />
          </span>
          <span className="text-slate-200">
            {badge.label}
            {badge.granted && (
              <span className="text-amber-300"> (from an effect)</span>
            )}
          </span>
        </li>
      ))}
    </ul>
  );
}
