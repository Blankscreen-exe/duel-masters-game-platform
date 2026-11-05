import type { ReactNode } from 'react';
import { CardImage } from '@/components/cards/card-image';
import { CARD_BACK } from '@/lib/card-backs';

/** A card shown on the how to play page: its picture, or its name while it has none. */
export interface ShownCard {
  readonly name: string;
  readonly imageUrl?: string | undefined;
}

/** The width of a card picture, smaller on phones so a row of them fits. */
export type CardSize = 'xs' | 'sm' | 'md' | 'lg';

const WIDTHS: Readonly<Record<CardSize, string>> = {
  xs: 'w-10 sm:w-14',
  sm: 'w-16 sm:w-24',
  md: 'w-24 sm:w-32',
  lg: 'w-56 sm:w-72',
};

/**
 * A card picture, upright or tapped (turned sideways, as the board shows a card that was used).
 * A tapped card keeps the room of a card on its side, so rows of cards line up.
 */
export function CardPic({
  card,
  size = 'sm',
  tapped = false,
}: {
  readonly card: ShownCard;
  readonly size?: CardSize;
  readonly tapped?: boolean;
}) {
  const picture = <CardImage src={card.imageUrl} name={card.name} />;

  if (!tapped) {
    return <div className={`shrink-0 ${WIDTHS[size]}`}>{picture}</div>;
  }

  // Turned a quarter, the card is as wide as its height was: a box of the card's height, on its
  // side, with the card at 72% (491 / 682) of the box's width standing in its middle.
  return (
    <div
      className={`grid shrink-0 place-items-center ${TAPPED_BOXES[size]}`}
      title={`${card.name}, tapped`}
    >
      <div className="w-[72%] rotate-90">{picture}</div>
    </div>
  );
}

const TAPPED_BOXES: Readonly<Record<CardSize, string>> = {
  xs: 'aspect-[682/491] w-14 sm:w-20',
  sm: 'aspect-[682/491] w-22 sm:w-33',
  md: 'aspect-[682/491] w-33 sm:w-44',
  lg: 'aspect-[682/491] w-78 sm:w-100',
};

/** The back of a card, as shields and the deck show it. */
export function CardBack({ size = 'sm' }: { readonly size?: CardSize }) {
  return (
    <div className={`shrink-0 ${WIDTHS[size]}`}>
      {/* eslint-disable-next-line @next/next/no-img-element -- a small file from public/ */}
      <img
        src={CARD_BACK}
        alt="A face-down card"
        className="aspect-[491/682] w-full rounded-[4.5%] object-cover shadow-md ring-1 ring-black/40"
      />
    </div>
  );
}

/** A picture with a label under it, and a line of detail under the label. */
export function Captioned({
  label,
  detail,
  children,
  className = '',
}: {
  readonly label: ReactNode;
  readonly detail?: ReactNode;
  readonly children: ReactNode;
  readonly className?: string;
}) {
  return (
    <figure className={`flex flex-col items-center gap-2 text-center ${className}`}>
      {children}
      <figcaption className="max-w-48">
        <span className="block font-display text-base font-bold tracking-wide text-white">
          {label}
        </span>
        {detail && <span className="block text-sm text-slate-400">{detail}</span>}
      </figcaption>
    </figure>
  );
}

/**
 * A spot on or just beside a card picture, as a share of its width and height (below 0 or above
 * 100 is outside the card), with a number for the key.
 */
export interface CardCallout {
  readonly x: number;
  readonly y: number;
  readonly label: string;
  readonly detail: string;
}

/** A large card with numbered markers on its parts, and a key naming each. */
export function AnnotatedCard({
  card,
  callouts,
}: {
  readonly card: ShownCard;
  readonly callouts: readonly CardCallout[];
}) {
  return (
    <div className="flex flex-col items-center gap-6 sm:flex-row sm:items-start">
      {/* Room around the card for the markers on its edges. */}
      <div className="relative mx-6 my-5 shrink-0">
        <CardPic card={card} size="lg" />
        {callouts.map((callout, index) => (
          <span
            key={callout.label}
            aria-hidden
            className="absolute flex size-7 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-accent font-display text-sm font-bold text-slate-950 shadow-[0_0_0_3px_rgb(2_6_23/0.85),0_0_14px_rgb(56_189_248/0.8)]"
            style={{ left: `${String(callout.x)}%`, top: `${String(callout.y)}%` }}
          >
            {index + 1}
          </span>
        ))}
      </div>
      <ol className="grid w-full gap-3 self-center">
        {callouts.map((callout, index) => (
          <li key={callout.label} className="flex items-start gap-3">
            <span
              aria-hidden
              className="flex size-7 shrink-0 items-center justify-center rounded-full bg-accent font-display text-sm font-bold text-slate-950"
            >
              {index + 1}
            </span>
            <span>
              <span className="font-display text-lg font-bold tracking-wide text-white">
                {callout.label}
              </span>
              <span className="block text-sm text-slate-400">{callout.detail}</span>
            </span>
          </li>
        ))}
      </ol>
    </div>
  );
}

/** A zone of the board: a dashed frame with its name on the edge and what it is for below. */
export function BoardZone({
  name,
  note,
  children,
  className = '',
}: {
  readonly name: string;
  readonly note: string;
  readonly children: ReactNode;
  readonly className?: string;
}) {
  return (
    <div
      className={`relative rounded-lg border border-dashed border-slate-500/60 bg-white/[0.02] px-3 pt-5 pb-2 ${className}`}
    >
      <span className="absolute -top-3 left-3 bg-panel px-2 font-display text-sm font-bold tracking-widest text-accent uppercase">
        {name}
      </span>
      <div className="flex flex-wrap items-center gap-2">{children}</div>
      <p className="mt-2 text-xs text-slate-400">{note}</p>
    </div>
  );
}

/**
 * An arrow between pictures, pointing down on phones where the pictures stack, unless the row
 * stays a row there too.
 */
export function FlowArrow({
  label,
  row = false,
}: {
  readonly label?: string;
  readonly row?: boolean;
}) {
  return (
    <div className="flex shrink-0 flex-col items-center gap-1 text-accent" aria-hidden>
      <span className={`text-3xl leading-none ${row ? 'hidden' : 'sm:hidden'}`}>↓</span>
      <span className={`text-3xl leading-none ${row ? '' : 'hidden sm:inline'}`}>→</span>
      {label && (
        <span className="max-w-28 text-center text-xs font-semibold tracking-wide text-slate-300 uppercase">
          {label}
        </span>
      )}
    </div>
  );
}

/**
 * A row of pictures and arrows. On phones it stacks, unless `row` keeps small pictures side by
 * side, as when a column of them would be needlessly long.
 */
export function Flow({
  children,
  row = false,
}: {
  readonly children: ReactNode;
  readonly row?: boolean;
}) {
  return (
    <div
      className={`flex items-center justify-center gap-4 sm:flex-row sm:gap-6 ${row ? 'flex-row' : 'flex-col'}`}
    >
      {children}
    </div>
  );
}

/** A big symbol between pictures, such as the versus of a battle. */
export function Versus({ children }: { readonly children: ReactNode }) {
  return (
    <span
      aria-hidden
      className="font-display text-3xl font-bold tracking-widest text-slate-500 uppercase"
    >
      {children}
    </span>
  );
}

/** One of the board's buttons, drawn as the board shows it, to point at in the text. */
export function BoardButton({
  children,
  tone = 'accent',
}: {
  readonly children: ReactNode;
  readonly tone?: 'accent' | 'quiet';
}) {
  return (
    <span
      className={`inline-flex items-center rounded px-2.5 py-1 font-display text-sm font-bold tracking-wide whitespace-nowrap uppercase ${
        tone === 'accent'
          ? 'bg-accent/20 text-accent ring-1 ring-accent/50'
          : 'bg-slate-700/60 text-slate-200 ring-1 ring-slate-500/50'
      }`}
    >
      {children}
    </span>
  );
}
