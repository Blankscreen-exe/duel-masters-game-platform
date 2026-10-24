/** One piece of a shattered shield. Positions are shares of the shield's box,
 * so the same pieces fit a shield of any size. */
export interface Shard {
  /** The piece's outline, as a CSS clip-path polygon. */
  clipPath: string;
  /** Where the piece's centre starts. */
  centre: { x: number; y: number };
  /** How far the piece flies, as a share of the box's width and height. */
  dx: number;
  dy: number;
  /** How far the piece turns while it flies, in degrees. */
  rotate: number;
  /** How long after the break the piece starts to fly, in seconds. */
  delay: number;
}

/** Points around the edge of the shield, in percent of its box, clockwise
 * from the top left corner. The cracks run from the point of impact to each. */
const RIM: readonly (readonly [number, number])[] = [
  [0, 0],
  [28, 0],
  [61, 0],
  [100, 0],
  [100, 27],
  [100, 59],
  [100, 100],
  [70, 100],
  [36, 100],
  [0, 100],
  [0, 72],
  [0, 35],
];

/** How far the point of impact may sit from the centre, in percent. */
const IMPACT_SPREAD = 12;
const MIN_FLIGHT = 0.45;
const EXTRA_FLIGHT = 0.5;
const MIN_TURN = 60;
const EXTRA_TURN = 200;
const MAX_DELAY = 0.05;

export interface Shattering {
  /** Where the shield was struck, as shares of its box. */
  impact: { x: number; y: number };
  shards: Shard[];
}

/** Cuts a shield into pieces that burst out from a point of impact. The same
 * seed always breaks the same way, and different shields break differently. */
export function shatter(seed: string): Shattering {
  const random = seededRandom(seed);
  const impact = {
    x: 50 + (random() * 2 - 1) * IMPACT_SPREAD,
    y: 45 + (random() * 2 - 1) * IMPACT_SPREAD,
  };

  const shards = RIM.map((point, index) => {
    const next = RIM[(index + 1) % RIM.length];
    const corners = [[impact.x, impact.y], point, next];
    const centre = {
      x: (impact.x + point[0] + next[0]) / 3,
      y: (impact.y + point[1] + next[1]) / 3,
    };
    const away = { x: centre.x - impact.x, y: centre.y - impact.y };
    const length = Math.hypot(away.x, away.y) || 1;
    const flight = MIN_FLIGHT + random() * EXTRA_FLIGHT;
    const turn = (MIN_TURN + random() * EXTRA_TURN) * (random() < 0.5 ? -1 : 1);

    return {
      clipPath: `polygon(${corners.map(([x, y]) => `${round(x)}% ${round(y)}%`).join(", ")})`,
      centre: { x: centre.x / 100, y: centre.y / 100 },
      dx: (away.x / length) * flight,
      dy: (away.y / length) * flight,
      rotate: Math.round(turn),
      delay: round(random() * MAX_DELAY, 3),
    };
  });

  return { impact: { x: impact.x / 100, y: impact.y / 100 }, shards };
}

function round(value: number, places = 2): number {
  const factor = 10 ** places;
  return Math.round(value * factor) / factor;
}

/** A small, fast generator of repeatable numbers between 0 and 1. */
function seededRandom(seed: string): () => number {
  let state = 2166136261;
  for (let i = 0; i < seed.length; i++) {
    state = Math.imul(state ^ seed.charCodeAt(i), 16777619);
  }

  return () => {
    state = (state + 0x6d2b79f5) | 0;
    let t = Math.imul(state ^ (state >>> 15), 1 | state);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}
