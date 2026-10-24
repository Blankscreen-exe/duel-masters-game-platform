import type { IconName } from "./icons";
import type { ChatCard } from "./types";

/** What a game log line reports, which picks its icon. */
export type LogKind =
  | "turnStart"
  | "turnEnd"
  | "summon"
  | "spell"
  | "shieldBreak"
  | "draw"
  | "shuffle"
  | "discard"
  | "destroy"
  | "toHand"
  | "toMana"
  | "toGraveyard"
  | "toShields"
  | "untap"
  | "tap"
  | "boost"
  | "effect";

export interface LogLook {
  icon: IconName;
  /** The icon's colour. */
  className: string;
  /** What the icon means, for screen readers and a tooltip. */
  label: string;
}

const LOG_LOOKS: Record<LogKind, LogLook> = {
  turnStart: { icon: "play", className: "text-sky-300", label: "Turn start" },
  turnEnd: { icon: "flag", className: "text-gray-300", label: "Turn end" },
  summon: { icon: "sparkle", className: "text-amber-300", label: "Summon" },
  spell: { icon: "wand", className: "text-violet-300", label: "Spell" },
  shieldBreak: {
    icon: "breaker",
    className: "text-red-300",
    label: "Shield break",
  },
  draw: { icon: "draw", className: "text-sky-300", label: "Draw" },
  shuffle: { icon: "shuffle", className: "text-gray-300", label: "Shuffle" },
  discard: { icon: "trash", className: "text-orange-300", label: "Discard" },
  destroy: { icon: "burst", className: "text-red-300", label: "Destroyed" },
  toHand: { icon: "toHand", className: "text-sky-300", label: "To hand" },
  toMana: { icon: "droplet", className: "text-emerald-300", label: "To mana" },
  toGraveyard: {
    icon: "tombstone",
    className: "text-gray-300",
    label: "To graveyard",
  },
  toShields: { icon: "shield", className: "text-sky-300", label: "To shields" },
  untap: { icon: "untap", className: "text-emerald-300", label: "Untap" },
  tap: { icon: "tapAbility", className: "text-emerald-300", label: "Tap" },
  boost: { icon: "arrowUp", className: "text-amber-300", label: "Power up" },
  effect: { icon: "dot", className: "text-gray-400", label: "Effect" },
};

export function logLook(kind: LogKind): LogLook {
  return LOG_LOOKS[kind];
}

const DESTINATIONS: Record<string, LogKind> = {
  hand: "toHand",
  mana: "toMana",
  manazone: "toMana",
  "mana zone": "toMana",
  mz: "toMana",
  graveyard: "toGraveyard",
  gy: "toGraveyard",
  shields: "toShields",
  shieldzone: "toShields",
  "shield zone": "toShields",
  deck: "shuffle",
  battlezone: "summon",
  "battle zone": "summon",
};

/** The zone a card was moved "to", "into" or put "in", as the server writes
 * it: "to Player2's hand", "into the shieldzone", "from MZ to GY". */
const DESTINATION =
  /\b(?:to|into|in)\s+(?:(?:the|its owner's|your|their|his|\S+'s)\s+)?(hand|mana ?zone|mana|mz|graveyard|gy|shield ?zone|shields|deck|battle ?zone)\b/i;

function destination(text: string): LogKind | null {
  const zone = DESTINATION.exec(text)?.[1].toLowerCase();

  return zone ? (DESTINATIONS[zone] ?? null) : null;
}

/** What a game log line reports, read from the server's wording. Where a card
 * went says the most, except that a destroyed card is reported as destroyed
 * unless something happened to it instead. */
export function logKind(text: string): LogKind {
  if (/^your turn\b/i.test(text)) return "turnStart";
  if (/\bended their turn\b/i.test(text)) return "turnEnd";
  if (/\bsummoned\b/i.test(text)) return "summon";
  if (/\bcast(?:ed|s)?\b/i.test(text)) return "spell";
  if (/\bshields? (?:was|were) broken\b/i.test(text)) return "shieldBreak";
  if (/\bdrew\b/i.test(text)) return "draw";
  if (/\bdiscard/i.test(text)) return "discard";

  const to = destination(text);

  if (to && /\binstead of\b/i.test(text)) return to;
  if (/\bshuffled?\b/i.test(text)) return "shuffle";
  if (/\bdest(?:ro|or)yed\b/i.test(text)) return "destroy";
  if (to) return to;
  if (/\bretrieved\b/i.test(text)) return "toHand";
  if (/\buntap/i.test(text)) return "untap";
  if (/\btap(?:ped)?\b|\btap (?:ability|effect)\b/i.test(text)) return "tap";
  if (/\bwas given\b|\+\d+ power\b/i.test(text)) return "boost";

  return "effect";
}

/** A piece of a log line: plain text, or a card it names. */
export type LogSegment =
  { text: string; card?: undefined } | { text: string; card: ChatCard };

/** Splits a log line into its text and the cards it names, which the server
 * lists in the order the line names them. A name the line does not contain
 * where expected is left as text. */
export function logSegments(
  text: string,
  cards: readonly ChatCard[] = [],
): LogSegment[] {
  const segments: LogSegment[] = [];
  let from = 0;

  for (const card of cards) {
    const at = text.indexOf(card.name, from);

    if (at < 0) {
      continue;
    }

    if (at > from) {
      segments.push({ text: text.slice(from, at) });
    }

    segments.push({ text: card.name, card });
    from = at + card.name.length;
  }

  if (from < text.length) {
    segments.push({ text: text.slice(from) });
  }

  return segments;
}

/** Whose side a named card is on, for its colour. */
export type CardSide = "mine" | "theirs" | "unknown";

export function cardSide(
  card: ChatCard,
  players: { me: string; opponent: string } | undefined,
): CardSide {
  if (!card.owner || !players) {
    return "unknown";
  }

  if (card.owner === players.me) {
    return "mine";
  }

  return card.owner === players.opponent ? "theirs" : "unknown";
}
