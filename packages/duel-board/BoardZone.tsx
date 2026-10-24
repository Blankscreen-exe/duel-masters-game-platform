import type { ReactNode } from "react";
import { useBoardAnchor } from "./BoardMotion";
import { CLIPS_CARDS_PROPS } from "./cardPositions";

export type DragZone =
  | "hand"
  | "myPlayzone"
  | "opponentPlayzone"
  | "myManazone"
  | "opponentManazone"
  | "myShieldzone"
  | "opponentShieldzone";

/** Whether dropping the dragged card on a zone would be accepted. */
export type DropHighlight = "green" | "red" | null;

const DROP_HIGHLIGHT_CLASSES: Record<"green" | "red", string> = {
  green: "bg-green-500/30",
  red: "bg-red-500/30",
};

interface BoardZoneProps {
  /** Names the zone for the drag and drop hit test, which finds zones by their
   * data-dropzone attribute. */
  zone: DragZone;
  /** Names the row as the place cards fly to or from when their own element
   * is not on screen. See anchorId. */
  anchor: string;
  /** The zone's share of the board's height, e.g. "h-[20%]". */
  heightClassName: string;
  dropHighlight?: DropHighlight;
  /** Padding around the row of cards. */
  rowClassName?: string;
  /** Centres the cards while they fit. A row wider than the zone still
   * scrolls from its first card, so none is cut off. */
  centered?: boolean;
  children: ReactNode;
}

/** One horizontal strip of the board holding a single zone's cards. A row
 * wider than the board scrolls sideways instead of squeezing the cards. */
export function BoardZone({
  zone,
  anchor,
  heightClassName,
  dropHighlight = null,
  rowClassName = "p-1",
  centered = false,
  children,
}: BoardZoneProps) {
  const rowRef = useBoardAnchor<HTMLDivElement>(anchor);

  return (
    <div className={`relative w-full ${heightClassName}`} data-dropzone={zone}>
      {dropHighlight && (
        <div
          className={`pointer-events-none absolute inset-0 z-20 ${DROP_HIGHLIGHT_CLASSES[dropHighlight]}`}
        />
      )}
      <div
        ref={rowRef}
        className="absolute inset-0 z-10 overflow-x-auto overflow-y-hidden"
        {...CLIPS_CARDS_PROPS}
      >
        <div
          className={`inline-flex h-full w-max gap-5 ${
            centered ? "min-w-full justify-center" : "justify-start"
          } ${rowClassName}`}
        >
          {children}
        </div>
      </div>
    </div>
  );
}
