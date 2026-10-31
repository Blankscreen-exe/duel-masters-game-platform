import { useEffect, useId, useState } from "react";
import { AnimatePresence, LayoutGroup, motion } from "motion/react";
import { displayPhase, TURN_PHASES, turnHint, turnNumber } from "./turnPhases";
import type { MatchState } from "./types";

const HIGHLIGHT_EASE = [0.4, 0, 0.2, 1] as const;

/** How long the hint stays up when it pops up below the bar. */
const HINT_BUBBLE_MS = 3000;

/** Shows how far the current turn has got, between the two halves of the
 * board. The server enforces the order of a turn, so this is what tells a
 * player why summoning stops working once they have attacked. */
export function PhaseBar({
  state,
  onEndTurn,
  compact = false,
  slim = false,
}: {
  state: MatchState;
  /** Shows an End turn button at the end of the bar when given. */
  onEndTurn?: () => void;
  /** A narrow screen has no room for the hint beside the bar, so it pops up
   * below it whenever it changes, and again when the bar is tapped. */
  compact?: boolean;
  /** Takes less height, for a phone on its side. */
  slim?: boolean;
}) {
  // Scopes the sliding highlight to this duel, since a page may show several.
  const layoutGroupId = useId();
  const currentPhase = displayPhase(state);
  const hint = turnHint(state);
  const turn = turnNumber(state);
  const [hintAskedAt, setHintAskedAt] = useState(0);
  const [bubbleShown, setBubbleShown] = useState(false);

  useEffect(() => {
    if (!compact || !hint) {
      setBubbleShown(false);
      return;
    }

    setBubbleShown(true);
    const timeout = window.setTimeout(
      () => setBubbleShown(false),
      HINT_BUBBLE_MS,
    );
    return () => window.clearTimeout(timeout);
  }, [compact, hint, hintAskedAt]);

  // Before the first turn there is no phase to show, but End turn must not
  // disappear with it.
  if (!currentPhase && !onEndTurn) {
    return null;
  }

  const currentIndex = TURN_PHASES.findIndex(
    (info) => info.phase === currentPhase,
  );
  const highlightClassName = state.spectator
    ? "bg-slate-500"
    : state.myTurn
      ? "bg-action"
      : "bg-red-700";

  return (
    // Narrow phones only fit the turn beside the phases with tighter spacing.
    <div
      className={`relative flex shrink-0 items-center justify-center gap-2 px-2 sm:gap-3 ${
        slim ? "h-6" : "h-10"
      }`}
    >
      {turn && (
        <span
          title="Turns so far this duel, counting both players' turns"
          className={`bz-chip flex shrink-0 items-baseline gap-1 font-display font-semibold uppercase leading-none tracking-wide text-slate-400 ${
            slim ? "px-2.5 py-1 text-[11px]" : "px-2 py-2 text-xs sm:px-3"
          }`}
        >
          Turn
          <motion.span
            key={turn}
            className="font-bold tabular-nums text-white"
            initial={{ opacity: 0, y: -6 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25, ease: HIGHLIGHT_EASE }}
          >
            {turn}
          </motion.span>
        </span>
      )}
      {currentPhase && (
        <LayoutGroup id={layoutGroupId}>
          <ol
            aria-label="Turn phases"
            className={`bz-chip flex shrink-0 items-center gap-0.5 p-0.5 font-display font-semibold uppercase leading-none tracking-wide ${slim ? "text-[11px]" : "text-xs"} ${
              compact && hint ? "cursor-pointer" : ""
            }`}
            onClick={compact ? () => setHintAskedAt(Date.now()) : undefined}
          >
            {TURN_PHASES.map((info, index) => {
              const isCurrent = index === currentIndex;

              return (
                <li
                  key={info.phase}
                  title={info.description}
                  aria-current={isCurrent ? "step" : undefined}
                  className={`relative ${slim ? "px-2.5 py-0.5" : "px-2 py-1.5 sm:px-3"} ${
                    isCurrent
                      ? "text-white"
                      : index < currentIndex
                        ? "text-slate-600"
                        : "text-slate-300"
                  }`}
                >
                  {isCurrent && (
                    <motion.span
                      layoutId="current-phase"
                      className={`absolute inset-0 ${highlightClassName}`}
                      transition={{ duration: 0.3, ease: HIGHLIGHT_EASE }}
                    />
                  )}
                  <span className="relative">{info.label}</span>
                </li>
              );
            })}
          </ol>
        </LayoutGroup>
      )}
      {hint && !compact && (
        <span
          aria-live="polite"
          className="hidden min-w-0 truncate text-sm text-slate-300 sm:block"
        >
          {hint}
        </span>
      )}
      <AnimatePresence>
        {hint && compact && bubbleShown && (
          <motion.span
            key={hint}
            role="status"
            className="pointer-events-none absolute left-1/2 top-full z-[26] mt-1 w-max max-w-[90vw] -translate-x-1/2 bg-panel px-3 py-1.5 text-center text-xs text-slate-200 shadow-lg ring-1 ring-slate-500/40"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
          >
            {hint}
          </motion.span>
        )}
      </AnimatePresence>
      {onEndTurn && (
        <button
          type="button"
          onClick={onEndTurn}
          disabled={!state.myTurn}
          title={state.myTurn ? undefined : "It's not your turn"}
          className="bz-button bz-button-primary shrink-0 cursor-pointer px-4 py-1.5 text-xs leading-none"
        >
          End turn
        </button>
      )}
    </div>
  );
}
