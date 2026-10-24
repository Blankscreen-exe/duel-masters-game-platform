import type { ReactNode } from "react";
import type { BoardSide } from "./boardDiff";
import { useBoardAnchor, useIsCardHidden, useTrackedCard } from "./BoardMotion";
import { CARD_IMAGE_PLACEHOLDER, cardImageUrl } from "./cardImage";
import { anchorId } from "./flights";
import type { CardState } from "./types";

const PILE_CLASS_NAME =
  "relative h-[12vh] min-h-[40px] max-h-[70px] min-[1200px]:min-h-[60px] min-[1200px]:max-h-[110px] flex items-center justify-center";

const CARD_STYLE = { ...CARD_IMAGE_PLACEHOLDER, borderRadius: "5%" };

function PileLabel({ children }: { children: ReactNode }) {
  return (
    <p className="text-[clamp(0.6rem,1.2vh,0.85rem)] leading-tight text-white mb-[0.5vh] text-center">
      {children}
    </p>
  );
}

/** How many cards are in a hand the viewer cannot see. Cards played from that
 * hand fly from here. */
export function HandCount({ side, count }: { side: BoardSide; count: number }) {
  const ref = useBoardAnchor<HTMLDivElement>(anchorId(side, "hand"));

  return (
    <div ref={ref}>
      <PileLabel>Hand [{count}]</PileLabel>
    </div>
  );
}

export function DeckPile({ side, count }: { side: BoardSide; count: number }) {
  const ref = useBoardAnchor<HTMLDivElement>(anchorId(side, "deck"));

  return (
    <div>
      <PileLabel>Deck [{count}]</PileLabel>
      <div ref={ref} className={PILE_CLASS_NAME}>
        <img
          src={cardImageUrl()}
          alt="Deck back"
          className="h-full"
          style={CARD_STYLE}
        />
      </div>
    </div>
  );
}

/** The graveyard, showing its top card. Opening it lists every card in it. */
export function GraveyardPile({
  side,
  cards,
  onOpen,
}: {
  side: BoardSide;
  cards: CardState[];
  onOpen: () => void;
}) {
  const pileRef = useBoardAnchor<HTMLDivElement>(anchorId(side, "graveyard"));
  const topCard = cards.length > 0 ? cards[cards.length - 1] : null;
  const topCardRef = useTrackedCard<HTMLImageElement>(
    topCard?.virtualId,
    topCard?.uid ?? "",
    0,
  );
  const topCardHidden = useIsCardHidden(topCard?.virtualId);

  return (
    <div>
      <PileLabel>Graveyard [{cards.length}]</PileLabel>
      <div ref={pileRef} className={PILE_CLASS_NAME}>
        {topCard ? (
          <img
            ref={topCardRef}
            src={cardImageUrl(topCard.uid)}
            alt="Top graveyard card"
            className="h-full cursor-pointer hover:scale-105 transition-transform"
            style={{
              ...CARD_STYLE,
              visibility: topCardHidden ? "hidden" : undefined,
            }}
            onClick={onOpen}
            onContextMenu={(e) => {
              e.preventDefault();
              onOpen();
            }}
          />
        ) : (
          <img
            src={cardImageUrl()}
            alt="Empty graveyard"
            className="h-full opacity-30"
            style={CARD_STYLE}
          />
        )}
      </div>
    </div>
  );
}
