import type { IconName } from "./icons";
import type { CreatureStatus, MatchState } from "./types";

/** The picture on a status badge. */
export type StatusIcon = IconName;

/** The kind of status, which gives a badge its colour. */
export type StatusTone =
  "attack" | "defense" | "support" | "limit" | "sickness";

export interface StatusBadge {
  id: string;
  icon: StatusIcon;
  tone: StatusTone;
  /** What the status means, for a tooltip or for viewing the card. */
  label: string;
  /** Given by another card's effect rather than printed on the card. */
  granted: boolean;
}

interface StatusLook {
  icon: StatusIcon;
  tone: StatusTone;
  label: (value: number) => string;
}

/** How each status the server reports is shown, by the condition it is named
 * after. A status this client does not know is left out. */
const STATUS_LOOKS: Record<string, StatusLook> = {
  summoning_sickness: {
    tone: "sickness",
    icon: "sleep",
    label: () =>
      "Summoning sickness: can't attack or use a tap ability this turn",
  },
  power_attacker: {
    tone: "attack",
    icon: "plus",
    label: (value) => `Power attacker: +${value} power while attacking`,
  },
  doublebreaker: {
    tone: "attack",
    icon: "breaker",
    label: () => "Double breaker: breaks 2 shields",
  },
  triplebreaker: {
    tone: "attack",
    icon: "breaker",
    label: () => "Triple breaker: breaks 3 shields",
  },
  shieldbreak_modifier: {
    tone: "attack",
    icon: "breaker",
    label: (value) =>
      `Breaks ${value} more ${value === 1 ? "shield" : "shields"}`,
  },
  speed_attacker: {
    tone: "attack",
    icon: "bolt",
    label: () => "Speed attacker: can attack the turn it is summoned",
  },
  slayer: {
    tone: "attack",
    icon: "skull",
    label: () => "Slayer: destroys the creature it battles",
  },
  cant_be_blocked: {
    tone: "attack",
    icon: "unblockable",
    label: () => "Can't be blocked",
  },
  stealth: {
    tone: "attack",
    icon: "unblockable",
    label: () =>
      "Stealth: can't be blocked while the opponent has mana of its civilization",
  },
  attack_untapped: {
    tone: "attack",
    icon: "target",
    label: () => "Can attack untapped creatures",
  },
  blocker: {
    tone: "defense",
    icon: "orb",
    label: () => "Blocker: can stop an attack",
  },
  survivor: {
    tone: "support",
    icon: "trefoil",
    label: () => "Survivor: shares its abilities with its owner's survivors",
  },
  cant_be_attacked: {
    tone: "defense",
    icon: "lock",
    label: () => "Can't be attacked",
  },
  treated_as_tapped: {
    tone: "limit",
    icon: "tilted",
    label: () => "Can be attacked as if it were tapped",
  },
  cant_be_selected_by_opp: {
    tone: "defense",
    icon: "hidden",
    label: () => "Can't be chosen by the opponent",
  },
  cant_be_tapped_by_opp: {
    tone: "defense",
    icon: "untappable",
    label: () => "Can't be tapped by the opponent",
  },
  cant_attack_creatures: {
    tone: "limit",
    icon: "noAttackCreatures",
    label: () => "Can't attack creatures",
  },
  cant_attack_players: {
    tone: "limit",
    icon: "noAttackPlayers",
    label: () => "Can't attack players",
  },
  doesnt_untap: {
    tone: "limit",
    icon: "noUntap",
    label: () => "Doesn't untap",
  },
  destroy_after_battle: {
    tone: "limit",
    icon: "hourglass",
    label: () => "Destroyed after it battles",
  },
  tap_ability: {
    tone: "support",
    icon: "tapAbility",
    label: () => "Tap ability: can tap instead of attacking to use its ability",
  },
  silent_skill: {
    tone: "support",
    icon: "silentSkill",
    label: () =>
      "Silent skill: can stay tapped at the start of its owner's turn to use its ability",
  },
  wave_striker: {
    tone: "support",
    icon: "wave",
    label: () =>
      "Wave striker: its ability works while 2 or more other wave strikers are in the battle zone",
  },
};

/** The badges for a creature's statuses, in the order the server sent them. */
export function statusBadges(
  statuses: readonly CreatureStatus[] = [],
): StatusBadge[] {
  return statuses.flatMap((status) => {
    const look = STATUS_LOOKS[status.id];

    return look
      ? [
          {
            id: status.id,
            icon: look.icon,
            tone: look.tone,
            label: look.label(status.value ?? 0),
            granted: !!status.granted,
          },
        ]
      : [];
  });
}

/** The statuses a player wants to see on the board at a glance, most
 * important first: what decides whether it can be blocked or attacked, and
 * how it attacks. Summoning sickness comes last, as the veil already shows it.
 * The rest are listed when the card is viewed. */
const BOARD_PRIORITY: readonly string[] = [
  "blocker",
  "slayer",
  "cant_be_blocked",
  "stealth",
  "cant_be_attacked",
  "attack_untapped",
  "treated_as_tapped",
  "power_attacker",
  "cant_attack_creatures",
  "cant_attack_players",
  "survivor",
  "summoning_sickness",
];

/** How many badges fit above a card before the rest are counted instead. */
export const MAX_BOARD_BADGES = 3;

export interface BoardBadges {
  /** The most important badges, at most MAX_BOARD_BADGES of them. */
  shown: StatusBadge[];
  /** How many more there are, which viewing the card lists. */
  more: number;
}

/** The badges shown above a creature on the board, most important first. Two
 * statuses that look the same, such as can't be blocked and stealth, are shown
 * once, and as granted if either was. */
export function boardStatusBadges(
  statuses: readonly CreatureStatus[] = [],
): BoardBadges {
  const byIcon = new Map<StatusIcon, StatusBadge>();
  const ranked = statusBadges(statuses)
    .filter((badge) => BOARD_PRIORITY.includes(badge.id))
    .sort(
      (a, b) => BOARD_PRIORITY.indexOf(a.id) - BOARD_PRIORITY.indexOf(b.id),
    );

  for (const badge of ranked) {
    const shown = byIcon.get(badge.icon);
    byIcon.set(
      badge.icon,
      shown ? { ...shown, granted: shown.granted || badge.granted } : badge,
    );
  }

  const badges = [...byIcon.values()];

  return {
    shown: badges.slice(0, MAX_BOARD_BADGES),
    more: Math.max(0, badges.length - MAX_BOARD_BADGES),
  };
}

/** The statuses of a creature in either battle zone. */
export function findCreatureStatuses(
  state: MatchState,
  virtualId: string,
): CreatureStatus[] {
  return (
    state.me.statuses?.[virtualId] ?? state.opponent.statuses?.[virtualId] ?? []
  );
}
