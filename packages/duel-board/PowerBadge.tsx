import { useEffect } from "react";
import {
  animate,
  motion,
  useMotionValue,
  useReducedMotion,
  useTransform,
} from "motion/react";
import {
  lowerEdgeShare,
  powerReadout,
  type PowerReadout,
} from "./creaturePower";
import type { CreaturePower } from "./types";

const TONE_CLASSES: Record<PowerReadout["tone"], string> = {
  raised: "bg-green-600",
  lowered: "bg-red-600",
  printed: "bg-black/75",
  attacking: "bg-amber-600",
};

/** Space between the badge and the card's lower edge, as a share of the box. */
const EDGE_INSET = 0.03;

const COUNT_SECONDS = 0.6;

/** A creature's power, at the bottom of its card, with the power it would
 * attack with beside it when that is more. Kept upright, like a shield's
 * number, so it reads the same from either side of the board. */
export function PowerBadge({
  power,
  rotated,
  attacking,
}: {
  power: CreaturePower;
  /** Whether the card lies on its side, which brings its lower edge up. */
  rotated: boolean;
  /** Whether the creature is attacking right now. */
  attacking: boolean;
}) {
  const readout = powerReadout(power, attacking);
  const shown = useCountingNumber(readout.shown);

  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute left-1/2 flex -translate-x-1/2 items-center gap-1 whitespace-nowrap text-[clamp(0.75rem,1.9vh,1.2rem)] font-black leading-none tracking-tight text-white"
      style={{ bottom: `${(lowerEdgeShare(rotated) + EDGE_INSET) * 100}%` }}
    >
      <span
        className={`rounded-md px-2 py-1 shadow-[0_2px_6px_rgba(0,0,0,0.9)] ring-1 ring-white/40 ${TONE_CLASSES[readout.tone]}`}
      >
        {readout.tone === "attacking" && "⚔ "}
        {/* A counting number has to be the only thing in its element. */}
        <motion.span>{shown}</motion.span>
      </span>
    </div>
  );
}

/** A number that counts to each new value instead of jumping to it. */
function useCountingNumber(value: number) {
  const reduceMotion = useReducedMotion();
  const counted = useMotionValue(value);
  const rounded = useTransform(counted, (latest) => Math.round(latest));

  useEffect(() => {
    if (reduceMotion) {
      counted.set(value);
      return;
    }

    const counting = animate(counted, value, {
      duration: COUNT_SECONDS,
      ease: "easeOut",
    });
    return () => counting.stop();
  }, [counted, value, reduceMotion]);

  return rounded;
}
