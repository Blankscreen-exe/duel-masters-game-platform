import { useEffect, useState } from "react";
import { useIsCardHidden, useTrackedCard } from "./BoardMotion";
import { describePower } from "./creaturePower";
import { PowerBadge } from "./PowerBadge";
import { boardStatusBadges, statusBadges } from "./creatureStatus";
import { StatusBadges } from "./StatusBadges";
import type { CreaturePower, CreatureStatus } from "./types";
import {
  useCardPictures,
  MUTED_CARD_FILTER,
  SHIELD_TINT,
  FACE_DOWN_IMAGE_ID,
} from "./cardImage";

export interface CardProps {
  virtualId?: string;
  name?: string;
  imageId?: string;
  rotated?: boolean;
  flipped?: boolean;
  /** Small index drawn in the card's top right corner, used to number shields. */
  number?: number;
  interactable?: boolean;
  canAddToBattlezone?: boolean;
  canAddToManazone?: boolean;
  hasTapAbility?: boolean;
  selected?: boolean;
  onAddToBattlezone?: (virtualId: string) => void;
  onAddToManazone?: (virtualId: string) => void;
  onTapAbility?: (virtualId: string) => void;
  isDragging?: boolean;
  draggable?: boolean;
  onDragStart?: (e: React.PointerEvent) => void;
  onRightClick?: () => void;
  /** Opens what can be done with the card. A draggable card leaves this to
   * the board, which tells a tap apart from the start of a drag. */
  onActivate?: () => void;
  /** Shows the card greyed out, as used up mana is. */
  muted?: boolean;
  /** Draws the card as a shield, under a glassy blue tint. */
  shield?: boolean;
  /** A creature's power, shown at the bottom of the card. */
  power?: CreaturePower;
  /** Whether the creature is attacking right now, which its power shows. */
  attacking?: boolean;
  /** Veils a creature that cannot attack yet because it was summoned this
   * turn. */
  summoningSick?: boolean;
  /** A creature's abilities and limits. Those that decide attacking and
   * blocking are shown as badges above the card. */
  statuses?: CreatureStatus[];
}

/** How far a card is turned clockwise, in degrees. A tapped card lies on its
 * side, and a flipped one faces the other player. */
export function cardRotation(rotated: boolean, flipped: boolean): number {
  if (rotated) {
    return flipped ? -90 : 90;
  }

  return flipped ? 180 : 0;
}

/** A purple grey veil over a creature that cannot attack yet. */
const SUMMONING_SICK_VEIL = {
  borderRadius: "5%",
  background:
    "linear-gradient(160deg, rgba(120, 104, 160, 0.55), rgba(64, 58, 88, 0.62))",
} as const;

const ROTATION_CLASSES: Record<number, string> = {
  0: "",
  90: "rotate-90",
  [-90]: "-rotate-90",
  180: "rotate-180",
};

export function Card({
  virtualId,
  name,
  imageId,
  rotated = false,
  flipped = false,
  number,
  interactable = false,
  selected = false,
  canAddToBattlezone = true,
  canAddToManazone = true,
  hasTapAbility = false,
  onAddToBattlezone,
  onAddToManazone,
  onTapAbility,
  isDragging = false,
  draggable = false,
  onDragStart,
  onRightClick,
  onActivate,
  muted = false,
  shield = false,
  power,
  attacking = false,
  summoningSick = false,
  statuses,
}: CardProps) {
  const pictures = useCardPictures();
  const badges = boardStatusBadges(statuses);
  // Hovering a card tells everything about it, not only what fits above it.
  const title =
    [
      power ? describePower(power) : null,
      ...statusBadges(statuses).map((badge) => badge.label),
    ]
      .filter(Boolean)
      .join(". ") || undefined;
  const activatesOnClick = !draggable && !!onActivate;
  const rotation = cardRotation(rotated, flipped);
  const imgRef = useTrackedCard<HTMLImageElement>(
    virtualId,
    imageId || FACE_DOWN_IMAGE_ID,
    rotation,
    muted,
  );
  const hidden = useIsCardHidden(virtualId);
  const [horizontalMargin, setHorizontalMargin] = useState("2rem");

  useEffect(() => {
    const updateMargin = () => {
      if (imgRef.current && rotated) {
        const height = imgRef.current.offsetHeight;
        // Calculate margin proportional to card height
        // Use smaller margins for smaller cards, larger for bigger cards
        const margin = Math.max(12, Math.min(32, height * 0.15));
        setHorizontalMargin(`${margin}px`);
      }
    };

    updateMargin();
    window.addEventListener("resize", updateMargin);
    return () => window.removeEventListener("resize", updateMargin);
  }, [rotated]);

  // Pointer events give one stream for mouse, touch and pen. Binding mouse and
  // touch separately made a tap fire both paths, because the browser follows a
  // touch with compatibility mouse events, and the second path immediately
  // undid the selection the first one made.
  const handlePointerDown = (e: React.PointerEvent) => {
    if (!draggable || !onDragStart) {
      return;
    }

    // Only the mouse needs its default suppressed, to stop a press from
    // starting a native image drag or a text selection. Doing the same for
    // touch would take the gesture away from the browser before it has had a
    // chance to decide whether the finger is scrolling the row.
    if (e.pointerType === "mouse") {
      e.preventDefault();
    }

    onDragStart(e);
  };

  const handleContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    if (onRightClick) {
      onRightClick();
    }
  };

  return (
    <>
      <div
        className="group relative flex-shrink-0 h-full"
        // Hidden, not removed, while a copy of the card is drawn above the
        // board, so the row keeps its place for it.
        style={hidden ? { visibility: "hidden" } : undefined}
      >
        {/* Card image */}
        <img
          ref={imgRef}
          src={pictures.url(imageId)}
          alt={name || "Backside card"}
          draggable={false}
          className={`block h-full w-auto flex-shrink-0 ${
            interactable && !isDragging
              ? "cursor-grab"
              : activatesOnClick
                ? "cursor-pointer"
                : ""
          } ${ROTATION_CLASSES[rotation]} ${
            isDragging ? "opacity-0" : ""
          } ${
            selected ? "ring-1 ring-blue-100" : ""
          }`}
          onPointerDown={handlePointerDown}
          onContextMenu={handleContextMenu}
          onClick={activatesOnClick ? onActivate : undefined}
          title={title}
          style={{
            // Rows scroll sideways while every drop target sits above or below
            // them, so the two gestures split cleanly by direction: the browser
            // keeps horizontal panning, and a drag has to start vertically.
            // Without this a finger landing on one of your own cards could
            // never scroll the row, because the card owned the whole gesture.
            // When the browser claims the pan it fires pointercancel, which the
            // board listens for to drop the half-started drag.
            touchAction: draggable ? "pan-x" : "auto",
            ...pictures.placeholder,
            borderRadius: "5%",
            marginLeft: rotated ? horizontalMargin : undefined,
            marginRight: rotated ? horizontalMargin : undefined,
            filter: muted ? MUTED_CARD_FILTER : undefined,
          }}
        />

        {shield && (
          <div
            className="pointer-events-none absolute inset-0"
            style={SHIELD_TINT}
          />
        )}

        {summoningSick && (
          // Laid over the image itself, margins and turn included, so it
          // covers the card however it lies.
          <div
            className={`pointer-events-none absolute inset-y-0 ${ROTATION_CLASSES[rotation]}`}
            style={{
              ...SUMMONING_SICK_VEIL,
              left: rotated ? horizontalMargin : 0,
              right: rotated ? horizontalMargin : 0,
            }}
          />
        )}

        <StatusBadges badges={badges} rotated={rotated} />

        {power && (
          <PowerBadge power={power} rotated={rotated} attacking={attacking} />
        )}

        {/* Kept upright and in the same screen corner even when the card is
            flipped, so a shield's number stays readable from either side. */}
        {number !== undefined && (
          <span className="pointer-events-none absolute top-[2%] right-[10%] text-[clamp(0.55rem,1.1vh,0.8rem)] leading-none text-white drop-shadow-[0_1px_2px_rgba(0,0,0,0.9)]">
            {number}
          </span>
        )}
      </div>
    </>
  );
}
