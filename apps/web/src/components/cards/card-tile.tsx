import Link from 'next/link';
import type { Card } from '@/lib/api/client';
import { CardImage } from './card-image';

interface CardTileProps {
  readonly card: Card;
  readonly priority?: boolean;
}

/** A card in a list: its picture, name and cost, leading to its page. */
export function CardTile({ card, priority = false }: CardTileProps) {
  return (
    <Link
      href={`/cards/${encodeURIComponent(card.id)}`}
      className="group block transition hover:-translate-y-1"
    >
      <CardImage
        src={card.imageUrl}
        name={card.name}
        priority={priority}
        className="transition group-hover:shadow-[0_0_20px_rgb(56_189_248/0.35)]"
      />
      <span className="mt-1.5 flex items-start justify-between gap-2 text-xs">
        <span className="line-clamp-2 font-semibold text-slate-200 group-hover:text-white">
          {card.name}
        </span>
        {!card.playable && (
          <span
            title="The rules engine cannot play this card yet, so it cannot go in decks."
            className="shrink-0 text-[10px] tracking-wider text-slate-500 uppercase"
          >
            Not playable
          </span>
        )}
      </span>
    </Link>
  );
}
