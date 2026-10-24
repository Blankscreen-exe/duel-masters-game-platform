import { useEffect, useState } from "react";
import { motion, type PanInfo } from "motion/react";
import { useMeasureCard } from "./BoardMotion";
import { Button } from "./Button";
import type { CardAction, CardActionId } from "./cardActions";
import { CARD_IMAGE_PLACEHOLDER, cardImageUrl } from "./cardImage";
import {
  liftedBox,
  restingTransform,
  sheetLayout,
  type Viewport,
} from "./cardMenuLayout";
import type { ScreenBox } from "./cardPositions";

const LIFT_TRANSITION = { duration: 0.22, ease: [0.4, 0, 0.2, 1] } as const;

/** How far a sheet has to be dragged towards its edge to close it. */
const SHEET_DISMISS_DISTANCE = 80;

interface CardMenuProps {
  virtualId: string;
  imageId: string;
  name?: string;
  actions: CardAction[];
  /** "overlay" lays the actions over the lifted card itself. "sheet" shows the
   * card large and readable, with the actions in a sheet along the edge of the
   * screen a thumb reaches, for phones. */
  layout: "overlay" | "sheet";
  onAction: (id: CardActionId) => void;
  onClose: () => void;
  /** Called once the card has settled back into its row after closing. */
  onDropped: () => void;
}

/** What can be done with a card. The card lifts out of its row, turns upright
 * and grows, and closing puts it back where it came from. */
export function CardMenu({
  virtualId,
  imageId,
  name,
  actions,
  layout,
  onAction,
  onClose,
  onDropped,
}: CardMenuProps) {
  const measureCard = useMeasureCard();
  // Read once, while the card is still where the player tapped it.
  const [origin] = useState(() => measureCard(virtualId));
  const viewport: Viewport = {
    width: window.innerWidth,
    height: window.innerHeight,
  };
  const sheet =
    layout === "sheet"
      ? sheetLayout(viewport, {
          actions: actions.length,
          reasons: actions.filter((action) => action.disabledReason).length,
        })
      : null;
  const shown = sheet ? sheet.card : liftedBox(origin, viewport);

  useEffect(() => {
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [onClose]);

  const label = name ? `${name} actions` : "Card actions";
  const actionList = (
    <ActionList
      actions={actions}
      onAction={onAction}
      explainDisabled={!!sheet}
    />
  );

  return (
    <motion.div
      className={`fixed inset-0 z-[35] ${sheet ? "bg-black/70" : "bg-black/40"}`}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={LIFT_TRANSITION}
      onClick={onClose}
    >
      <LiftedCard
        origin={origin}
        shown={shown}
        imageId={imageId}
        name={name}
        role={sheet ? undefined : "dialog"}
        label={sheet ? undefined : label}
        onDropped={onDropped}
      >
        {!sheet && (
          <motion.div
            className="absolute inset-0 flex flex-col justify-end gap-2 overflow-hidden rounded-[5%] bg-gradient-to-t from-black/90 via-black/70 to-transparent p-3 text-white"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={LIFT_TRANSITION}
          >
            {actionList}
          </motion.div>
        )}
      </LiftedCard>

      {sheet && (
        <ActionSheet
          edge={sheet.edge}
          size={sheet.size}
          label={label}
          name={name}
          onClose={onClose}
        >
          {actionList}
        </ActionSheet>
      )}
    </motion.div>
  );
}

/** The card itself, lifted from its row to where the menu shows it. */
function LiftedCard({
  origin,
  shown,
  imageId,
  name,
  role,
  label,
  onDropped,
  children,
}: {
  origin: ScreenBox | null;
  shown: ScreenBox;
  imageId: string;
  name?: string;
  role?: "dialog";
  label?: string;
  onDropped: () => void;
  children?: React.ReactNode;
}) {
  const variants = {
    // Without a known origin the card just fades in and out where it is.
    resting: origin
      ? restingTransform(origin, shown)
      : { scale: 0.9, opacity: 0 },
    lifted: { x: 0, y: 0, scale: 1, rotate: 0, opacity: 1 },
  };

  return (
    <motion.div
      role={role}
      aria-label={label}
      className="absolute shadow-2xl"
      style={{
        left: shown.x - shown.width / 2,
        top: shown.y - shown.height / 2,
        width: shown.width,
        height: shown.height,
      }}
      variants={variants}
      initial="resting"
      animate="lifted"
      exit="resting"
      transition={LIFT_TRANSITION}
      onAnimationComplete={(definition) => {
        if (definition === "resting") {
          onDropped();
        }
      }}
      onClick={(event) => event.stopPropagation()}
    >
      <img
        src={cardImageUrl(imageId)}
        alt={name || "Card"}
        draggable={false}
        className="block h-full w-full"
        style={{ ...CARD_IMAGE_PLACEHOLDER, borderRadius: "5%" }}
      />
      {children}
    </motion.div>
  );
}

/** A panel of actions along the bottom of an upright phone, or the right of
 * one on its side. It can be dragged back towards its edge to close it. */
function ActionSheet({
  edge,
  size,
  label,
  name,
  onClose,
  children,
}: {
  edge: "bottom" | "right";
  size: number;
  label: string;
  name?: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  const fromBottom = edge === "bottom";
  const hidden = fromBottom ? { y: "100%" } : { x: "100%" };

  const closeIfDraggedAway = (_event: unknown, info: PanInfo) => {
    const distance = fromBottom ? info.offset.y : info.offset.x;

    if (distance > SHEET_DISMISS_DISTANCE) {
      onClose();
    }
  };

  return (
    <motion.div
      role="dialog"
      aria-label={label}
      className={`absolute flex flex-col gap-2 bg-gray-900 p-4 text-white shadow-2xl ${
        fromBottom
          ? "inset-x-0 bottom-0 rounded-t-2xl"
          : "inset-y-0 right-0 justify-center rounded-l-2xl"
      }`}
      style={{
        ...(fromBottom ? { maxHeight: size } : { width: size }),
        // Clear of a phone's home indicator and rounded corners.
        paddingBottom: "max(1rem, env(safe-area-inset-bottom))",
        paddingRight: fromBottom
          ? undefined
          : "max(1rem, env(safe-area-inset-right))",
      }}
      initial={hidden}
      animate={{ x: 0, y: 0 }}
      exit={hidden}
      transition={LIFT_TRANSITION}
      drag={fromBottom ? "y" : "x"}
      dragConstraints={{ top: 0, bottom: 0, left: 0, right: 0 }}
      dragElastic={
        fromBottom ? { top: 0, bottom: 0.6 } : { left: 0, right: 0.6 }
      }
      onDragEnd={closeIfDraggedAway}
      onClick={(event) => event.stopPropagation()}
    >
      {fromBottom && (
        <div className="mx-auto h-1 w-10 shrink-0 rounded-full bg-gray-600" />
      )}
      {name && (
        <p className="shrink-0 truncate text-base font-semibold">{name}</p>
      )}
      {children}
    </motion.div>
  );
}

function ActionList({
  actions,
  onAction,
  explainDisabled,
}: {
  actions: CardAction[];
  onAction: (id: CardActionId) => void;
  /** Writes out why an action is disabled, for screens without hover. */
  explainDisabled: boolean;
}) {
  const firstEnabled = actions.findIndex((action) => !action.disabledReason);

  return (
    <div className="flex min-h-0 flex-col gap-2 overflow-y-auto">
      {actions.map((action, index) => (
        <div key={action.id}>
          <Button
            onClick={() => onAction(action.id)}
            disabled={!!action.disabledReason}
            disabledTooltip={
              explainDisabled ? undefined : (action.disabledReason ?? undefined)
            }
            variant={action.id === "view" ? "gray" : "default"}
            size={explainDisabled ? "large" : "default"}
            autoFocus={index === firstEnabled}
          >
            {action.label}
          </Button>
          {explainDisabled && action.disabledReason && (
            <p className="mt-1 px-1 text-xs leading-tight text-gray-400">
              {action.disabledReason}
            </p>
          )}
        </div>
      ))}
    </div>
  );
}
