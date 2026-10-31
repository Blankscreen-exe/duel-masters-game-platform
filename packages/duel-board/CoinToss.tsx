import { motion, useReducedMotion } from "motion/react";
import { useEffect, useState } from "react";
import { Button } from "./Button";
import type { CoinSide, CoinTossState } from "./coinTossState";

/** How long the coin spins before it lands. */
const FLIP_SECONDS = 1.8;
/** Whole turns the coin makes on the way. */
const FLIP_TURNS = 6;

interface CoinTossProps {
  toss: CoinTossState;
  onCall: (side: CoinSide) => void;
  onChooseTurn: (playFirst: boolean) => void;
}

/** The coin toss that decides who plays first, over the board until the game
 * begins: someone calls heads or tails, the coin is flipped and lands where
 * the server decided, and its winner chooses to play first or second. */
export function CoinToss({ toss, onCall, onChooseTurn }: CoinTossProps) {
  const reduceMotion = useReducedMotion();
  // Once a choice is sent, the buttons wait for the server.
  const [sent, setSent] = useState(false);
  const [landed, setLanded] = useState(false);
  const outcome = toss.step === "tossed" ? toss.outcome : null;

  // Every new step of the toss can be answered again.
  useEffect(() => {
    setSent(false);
  }, [toss.step, toss.step === "tossed" && toss.mine]);

  useEffect(() => {
    setLanded(false);
    if (outcome === null) {
      return;
    }
    const timer = window.setTimeout(
      () => setLanded(true),
      reduceMotion ? 0 : FLIP_SECONDS * 1000,
    );
    return () => window.clearTimeout(timer);
  }, [outcome, reduceMotion]);

  return (
    <div
      role="dialog"
      aria-label="Coin toss"
      className="bz-backdrop fixed inset-0 z-40 flex flex-col items-center justify-center gap-6 p-6 text-center text-slate-200"
    >
      <Coin outcome={outcome} reduceMotion={!!reduceMotion} />

      <div aria-live="polite" className="max-w-sm space-y-4">
        {toss.step === "calling" ? (
          toss.mine ? (
            <>
              <p className="bz-title text-2xl">Call the coin toss</p>
              <p className="text-sm text-slate-400">
                Win it and you choose whether to play first.
              </p>
              <div className="flex justify-center gap-3">
                {(["heads", "tails"] as const).map((side) => (
                  <Button
                    key={side}
                    disabled={sent}
                    onClick={() => {
                      setSent(true);
                      onCall(side);
                    }}
                  >
                    {side === "heads" ? "Heads" : "Tails"}
                  </Button>
                ))}
              </div>
            </>
          ) : (
            <p className="text-lg">
              <strong>{toss.caller.username}</strong> is calling the coin toss…
            </p>
          )
        ) : landed ? (
          <>
            <p className="text-lg">
              {toss.caller.username} called {toss.call}. It is{" "}
              <strong>{toss.outcome}</strong>!
            </p>
            <p className="bz-title text-2xl">
              {toss.winner.username} won the toss
            </p>
            {toss.mine === true && (
              <div className="flex justify-center gap-3">
                <Button
                  disabled={sent}
                  onClick={() => {
                    setSent(true);
                    onChooseTurn(true);
                  }}
                >
                  Play first
                </Button>
                <Button
                  variant="gray"
                  disabled={sent}
                  onClick={() => {
                    setSent(true);
                    onChooseTurn(false);
                  }}
                >
                  Play second
                </Button>
              </div>
            )}
            {toss.mine === false && (
              <p className="text-sm text-slate-400">
                {toss.winner.username} is choosing who plays first…
              </p>
            )}
          </>
        ) : (
          <p className="text-lg">
            {toss.caller.username} called {toss.call}…
          </p>
        )}
      </div>
    </div>
  );
}

/** A gold coin, heads on one face and tails on the other. Before the toss it
 * turns slowly; once tossed it spins and lands on the outcome. */
function Coin({
  outcome,
  reduceMotion,
}: {
  outcome: CoinSide | null;
  reduceMotion: boolean;
}) {
  const landing = outcome === "tails" ? 180 : 0;

  return (
    <div className="h-32 w-32" style={{ perspective: 800 }}>
      <motion.div
        className="relative h-full w-full"
        style={{ transformStyle: "preserve-3d" }}
        initial={{ rotateY: 0 }}
        animate={
          outcome === null
            ? reduceMotion
              ? { rotateY: 0 }
              : { rotateY: [0, 360] }
            : { rotateY: (reduceMotion ? 0 : FLIP_TURNS * 360) + landing }
        }
        transition={
          outcome === null
            ? { duration: 4, repeat: Infinity, ease: "linear" }
            : { duration: reduceMotion ? 0 : FLIP_SECONDS, ease: [0.22, 1, 0.36, 1] }
        }
      >
        <CoinFace label="Heads" letter="H" />
        <CoinFace label="Tails" letter="T" back />
      </motion.div>
    </div>
  );
}

function CoinFace({
  label,
  letter,
  back = false,
}: {
  label: string;
  letter: string;
  back?: boolean;
}) {
  return (
    <div
      aria-label={label}
      className="absolute inset-0 flex items-center justify-center rounded-full border-4 border-yellow-200/80 bg-[radial-gradient(circle_at_35%_30%,#fef3c7,#f59e0b_55%,#b45309)] text-5xl font-bold text-yellow-900 shadow-[0_0_30px_rgba(245,158,11,0.45)]"
      style={{
        backfaceVisibility: "hidden",
        transform: back ? "rotateY(180deg)" : undefined,
      }}
    >
      {letter}
    </div>
  );
}
