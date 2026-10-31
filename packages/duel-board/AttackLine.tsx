import {
  describeAttack,
  type AttackEnd,
  type AttackNotice,
  type AttackOutcome,
} from "./attackNotice";
import { CARD_SIDE_CLASSES } from "./chatLog";
import { Icon } from "./icons";

const OTHER_SIDE = { mine: "theirs", theirs: "mine" } as const;

const OUTCOME_TONES: Record<AttackOutcome["tone"], string> = {
  win: "text-green-400",
  lose: "text-red-400",
  draw: "text-amber-300",
  neutral: "text-white",
};

/** One side of the attack: a name, with its power large beneath it. */
function End({
  end,
  sword = false,
  className,
}: {
  end: AttackEnd;
  sword?: boolean;
  className: string;
}) {
  return (
    <div className={`min-w-0 text-center ${className}`}>
      <p className="text-sm font-semibold break-words">{end.name}</p>
      {end.power !== undefined && (
        <p className="font-display text-3xl leading-tight font-bold">
          {sword && "⚔ "}
          {end.power}
        </p>
      )}
    </div>
  );
}

/** What the attack will do. */
function Outcome({ outcome }: { outcome: AttackOutcome }) {
  const { tone, headline, destroyed, blockers } = outcome;

  return (
    <div className="mt-3 border-t border-slate-600/40 pt-3 text-center">
      <p
        className={`font-display text-xl font-bold tracking-wide uppercase ${OUTCOME_TONES[tone]}`}
      >
        {headline}
      </p>
      {destroyed.length > 0 && (
        <p className="mt-1 text-sm text-slate-200">
          Destroyed: {destroyed.join(" and ")}
        </p>
      )}
      {blockers.length > 0 && (
        <p className="mt-2 text-xs text-amber-300">
          Unless your opponent blocks with {blockers.join(" or ")}
        </p>
      )}
    </div>
  );
}

/** An attack: the attacker and its power, an arrow, and what it attacks, each
 * in the colour of the side it is on: blue for the viewer, red for the
 * opponent. Below it, what the attack will do when that is known. */
export function AttackLine({
  notice,
  className = "",
}: {
  notice: AttackNotice;
  className?: string;
}) {
  const { side, attacker, target, outcome } = notice;

  return (
    <div
      className={`bg-field p-3 ring-1 ring-slate-600/40 ring-inset ${className}`}
    >
      <div
        role="img"
        aria-label={describeAttack(notice)}
        className="grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2"
      >
        <End end={attacker} sword className={CARD_SIDE_CLASSES[side]} />
        <Icon
          name="arrowRight"
          className={`h-8 w-8 ${CARD_SIDE_CLASSES[side]}`}
        />
        <End end={target} className={CARD_SIDE_CLASSES[OTHER_SIDE[side]]} />
      </div>
      {outcome && <Outcome outcome={outcome} />}
    </div>
  );
}
