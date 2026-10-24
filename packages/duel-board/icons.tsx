import type { ReactNode } from "react";

/** A line drawn through an icon to say something cannot happen. */
const STRUCK_THROUGH = <path d="M3 21L21 3" />;

const SHIELD = "M12 3l7 3v5c0 5-3.5 8.5-7 10-3.5-1.5-7-5-7-10V6z";

/** The icons the duel board draws, on a 24 unit grid with round strokes in the
 * current colour. The ability symbols printed on cards are redrawn here, so
 * they stay sharp at any size: blocker's orb, survivor's trefoil, tap
 * ability's curled arrow, silent skill's hooded head and wave striker's
 * striped sphere. */
const ICONS = {
  sleep: (
    <text
      x="12"
      y="16.5"
      textAnchor="middle"
      fontSize="13"
      fontWeight="900"
      fontStyle="italic"
      fill="currentColor"
      stroke="none"
    >
      Zz
    </text>
  ),
  plus: <path d="M12 5v14M5 12h14" strokeWidth={3} />,
  breaker: <path d={`${SHIELD}M12.5 7l-2 4 3 2-2 4`} />,
  bolt: <path d="M13 2L4 14h7l-1 8 9-12h-7z" />,
  skull: (
    <>
      <path d="M12 3a7 7 0 00-7 7c0 2.6 1.4 4 2.5 4.7V18h9v-3.3C17.6 14 19 12.6 19 10a7 7 0 00-7-7zM10 18v3M14 18v3" />
      <circle cx="9.5" cy="10.5" r="1.2" />
      <circle cx="14.5" cy="10.5" r="1.2" />
    </>
  ),
  unblockable: <path d="M3 12h13M12 8l4 4-4 4M20 4v16" />,
  target: (
    <>
      <circle cx="12" cy="12" r="7" />
      <circle cx="12" cy="12" r="2" />
      <path d="M12 2v3M12 19v3M2 12h3M19 12h3" />
    </>
  ),
  orb: (
    <>
      <circle cx="12" cy="12" r="8.5" fill="currentColor" fillOpacity={0.3} />
      <circle cx="9" cy="9" r="2.2" fill="currentColor" stroke="none" />
    </>
  ),
  trefoil: (
    <>
      <circle cx="12" cy="7.5" r="4.3" />
      <circle cx="7.6" cy="15" r="4.3" />
      <circle cx="16.4" cy="15" r="4.3" />
      <circle cx="12" cy="12.4" r="1.4" fill="currentColor" stroke="none" />
    </>
  ),
  lock: (
    <>
      <rect x="5" y="11" width="14" height="10" rx="2" />
      <path d="M8 11V8a4 4 0 018 0v3" />
    </>
  ),
  tilted: (
    <>
      <rect
        x="9"
        y="4"
        width="10"
        height="15"
        rx="2"
        transform="rotate(40 14 11.5)"
      />
      <path d="M3 20a9 9 0 014-7.5M7 12.5l.2 3.3M7 12.5l-3.2.8" />
    </>
  ),
  hidden: (
    <path d="M3 3l18 18M10.6 6.1A9.8 9.8 0 0112 6c5 0 9 6 9 6a17 17 0 01-3.2 3.8M6.6 7.6C4.3 9.2 3 12 3 12s4 6 9 6a8.7 8.7 0 004.4-1.2" />
  ),
  untappable: (
    <>
      <rect x="7" y="3" width="10" height="18" rx="2" />
      {STRUCK_THROUGH}
    </>
  ),
  noAttackCreatures: (
    <>
      <path d="M6.5 5c2 4 2 9 0 14M11.5 4c2 5 2 11 0 16M16.5 5c2 4 2 9 0 14" />
      {STRUCK_THROUGH}
    </>
  ),
  noAttackPlayers: (
    <>
      <path d={SHIELD} />
      {STRUCK_THROUGH}
    </>
  ),
  noUntap: <path d="M4 12a8 8 0 0113.7-5.7L20 9M20 4v5h-5M4 4l16 16" />,
  hourglass: (
    <path d="M7 3h10M7 21h10M8 3c0 4 8 5 8 9s-8 5-8 9M16 3c0 4-8 5-8 9s8 5 8 9" />
  ),
  tapAbility: <path d="M12 4a8 8 0 108 8h-6M16.5 9.5L14 12l2.5 2.5" />,
  silentSkill: (
    <path d="M7 14a6 6 0 0112 0v4H7zM9.5 14h7M2 9.5h3M1.5 13.5h3M2 17.5h3" />
  ),
  wave: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M4.5 10c2.5-2 5 2 7.5 0s5-2 7.5 0M4.5 14.5c2.5-2 5 2 7.5 0s5-2 7.5 0" />
    </>
  ),
  shield: <path d={SHIELD} />,
  play: <path d="M8 5l11 7-11 7z" />,
  flag: <path d="M6 21V4M6 4h11l-2 4 2 4H6" />,
  sparkle: (
    <path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8zM18.5 16l.7 2 2 .7-2 .7-.7 2-.7-2-2-.7 2-.7z" />
  ),
  wand: <path d="M4 20L15 9M14 3v3M18.5 5.5l-2 2M21 10h-3M13 7l4 4" />,
  droplet: (
    <path d="M12 3c3.5 4.5 6 7.7 6 10.5a6 6 0 01-12 0C6 10.7 8.5 7.5 12 3z" />
  ),
  draw: (
    <>
      <rect x="6" y="3" width="12" height="17" rx="2" />
      <path d="M12 8v6M9 11.5l3 3 3-3" />
    </>
  ),
  burst: (
    <path d="M12 2l2 6 6-2-4 5 5 4-6 1 1 6-4-4-4 4 1-6-6-1 5-4-4-5 6 2z" />
  ),
  tombstone: <path d="M7 21V10a5 5 0 0110 0v11zM4 21h16M12 11v5M10 13h4" />,
  toHand: <path d="M9 14L4 9l5-5M4 9h10a6 6 0 010 12h-3" />,
  trash: <path d="M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14M10 11v6M14 11v6" />,
  shuffle: (
    <path d="M4 7h3c4 0 6 10 10 10h3M17 14l3 3-3 3M4 17h3c1.5 0 2.7-1.4 3.8-3.3M13.2 10.3C14.3 8.4 15.5 7 17 7h3M17 4l3 3-3 3" />
  ),
  untap: <path d="M20 12a8 8 0 11-3-6.2M20 4v5h-5" />,
  arrowUp: <path d="M12 19V5M6 11l6-6 6 6" />,
  dot: <circle cx="12" cy="12" r="3" fill="currentColor" stroke="none" />,
} satisfies Record<string, ReactNode>;

export type IconName = keyof typeof ICONS;

/** One of the board's icons, sized to the surrounding text. */
export function Icon({
  name,
  className = "h-[1em] w-[1em]",
}: {
  name: IconName;
  className?: string;
}) {
  return (
    <svg
      viewBox="0 0 24 24"
      className={className}
      fill="none"
      stroke="currentColor"
      strokeWidth={2.2}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {ICONS[name]}
    </svg>
  );
}
