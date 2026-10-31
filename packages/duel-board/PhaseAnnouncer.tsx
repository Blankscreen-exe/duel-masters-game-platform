import { useCallback, useEffect, useRef, useState } from "react";
import { motion, useReducedMotion } from "motion/react";
import {
  BANNER_ENTER_SECONDS,
  bannerHoldSeconds,
  bannerSeconds,
  phaseBanners,
  queueBanners,
  type PhaseBanner,
} from "./turnPhases";
import type { MatchState } from "./types";

/** The points of a banner's run, as shares of its length: in from the left,
 * held, then out to the right. */
function bannerTimes(banner: PhaseBanner): number[] {
  const total = bannerSeconds(banner);

  return [
    0,
    BANNER_ENTER_SECONDS / total,
    (BANNER_ENTER_SECONDS + bannerHoldSeconds(banner)) / total,
    1,
  ];
}

/** If a banner's end is somehow never reported, the next one follows anyway
 * this long after it should have finished. */
const BANNER_GRACE_MS = 500;

const TONE_CLASSES: Record<PhaseBanner["tone"], string> = {
  mine: "text-sky-300 drop-shadow-[0_0_14px_rgba(56,189,248,0.9)]",
  theirs: "text-red-300 drop-shadow-[0_0_14px_rgba(239,68,68,0.9)]",
  neutral: "text-slate-100 drop-shadow-[0_0_14px_rgba(255,255,255,0.6)]",
};

interface QueuedBanner extends PhaseBanner {
  key: number;
}

/** Announces each new turn and phase with a title that sweeps across the
 * middle of the board. It never takes a click, so play carries on beneath. */
export function PhaseAnnouncer({ state }: { state: MatchState }) {
  const [queue, setQueue] = useState<QueuedBanner[]>([]);
  const previous = useRef<MatchState | null>(null);
  const nextKey = useRef(0);

  useEffect(() => {
    const banners = phaseBanners(previous.current, state);
    previous.current = state;

    if (banners.length > 0) {
      setQueue((current) =>
        queueBanners(
          current,
          banners.map((banner) => ({ ...banner, key: nextKey.current++ })),
        ),
      );
    }
  }, [state]);

  const handleDone = useCallback((key: number) => {
    setQueue((current) =>
      current[0]?.key === key ? current.slice(1) : current,
    );
  }, []);

  const showing = queue[0];

  return showing ? (
    <Banner key={showing.key} banner={showing} onDone={handleDone} />
  ) : null;
}

function Banner({
  banner,
  onDone,
}: {
  banner: QueuedBanner;
  onDone: (key: number) => void;
}) {
  // Without motion the title fades in and out in the middle instead.
  const reduceMotion = useReducedMotion();
  const { key, kind, title, tone } = banner;

  useEffect(() => {
    const timeout = window.setTimeout(
      () => onDone(key),
      bannerSeconds(banner) * 1000 + BANNER_GRACE_MS,
    );
    return () => window.clearTimeout(timeout);
  }, [banner, key, onDone]);

  const transition = {
    duration: bannerSeconds(banner),
    times: bannerTimes(banner),
  };

  return (
    <div
      aria-live="polite"
      className="pointer-events-none fixed inset-x-0 top-1/2 z-[30] -translate-y-1/2 overflow-hidden"
    >
      <div className="relative py-3">
        {/* The band fades on its own, so the title stays solid all the way
            off the screen. */}
        <motion.div
          className="absolute inset-0 bg-gradient-to-r from-transparent via-black/60 to-transparent"
          initial={{ opacity: 0 }}
          animate={{ opacity: [0, 1, 1, 0] }}
          transition={transition}
        />
        <motion.p
          className={`relative whitespace-nowrap text-center font-display font-bold uppercase tracking-[0.2em] ${
            kind === "turn" ? "text-3xl sm:text-5xl" : "text-2xl sm:text-4xl"
          } ${TONE_CLASSES[tone]}`}
          initial={reduceMotion ? { opacity: 0 } : { x: "-100vw" }}
          animate={
            reduceMotion
              ? { opacity: [0, 1, 1, 0] }
              : { x: ["-100vw", "0vw", "0vw", "100vw"] }
          }
          transition={{
            ...transition,
            ease: ["easeOut", "linear", "easeIn"],
          }}
          onAnimationComplete={() => onDone(key)}
        >
          {title}
        </motion.p>
      </div>
    </div>
  );
}
