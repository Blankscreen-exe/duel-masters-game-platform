import {
  describeAttack,
  type AttackEnd,
  type AttackNotice,
} from "./attackNotice";
import { CARD_SIDE_CLASSES } from "./chatLog";
import { Icon } from "./icons";

const OTHER_SIDE = { mine: "theirs", theirs: "mine" } as const;

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

/** An attack: the attacker and its power, an arrow, and what it attacks, each
 * in the colour of the side it is on: blue for the viewer, red for the
 * opponent. */
export function AttackLine({
  notice,
  className = "",
}: {
  notice: AttackNotice;
  className?: string;
}) {
  const { side, attacker, target } = notice;

  return (
    <div
      role="img"
      aria-label={describeAttack(notice)}
      className={`grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2 bg-field p-3 ring-1 ring-slate-600/40 ring-inset ${className}`}
    >
      <End end={attacker} sword className={CARD_SIDE_CLASSES[side]} />
      <Icon
        name="arrowRight"
        className={`h-8 w-8 ${CARD_SIDE_CLASSES[side]}`}
      />
      <End end={target} className={CARD_SIDE_CLASSES[OTHER_SIDE[side]]} />
    </div>
  );
}
