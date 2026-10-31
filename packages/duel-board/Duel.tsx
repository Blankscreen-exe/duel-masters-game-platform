import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { AnimatePresence } from "motion/react";
import { useDuel } from "./useDuel";
import type { ActionAnswer } from "./answers";
import {
  ActionMessage,
  ActionWarningMessage,
  cardHasFlag,
  CardState,
  ChatCard,
  CreaturePower,
  CreatureStatus,
  DuelChatUserResolver,
  DuelChatUserTriggerRenderer,
  DuelFinishedMessage,
  MatchState,
  PLAYABLE_FLAG,
  ShieldState,
  SUMMONING_SICK_FLAG,
  TAPPED_FLAG,
} from "./types";
import { Card } from "./Card";
import { BoardZone, type DragZone, type DropHighlight } from "./BoardZone";
import { BoardMotion } from "./BoardMotion";
import { CLIPS_CARDS_PROPS } from "./cardPositions";
import { anchorId } from "./flights";
import { DeckPile, GraveyardPile, HandCount } from "./InfoPiles";
import { PhaseBar } from "./PhaseBar";
import { PhaseAnnouncer } from "./PhaseAnnouncer";
import { CardMenu } from "./CardMenu";
import { findCreaturePower } from "./creaturePower";
import { findCreatureStatuses, statusBadges } from "./creatureStatus";
import { STATUS_LANE_CLASS } from "./StatusBadges";
import {
  cardActions,
  findBoardCard,
  type BoardCard,
  type CardActionId,
} from "./cardActions";
import {
  attackForfeit,
  describeAttackForfeit,
  type AttackForfeit,
} from "./turnPhases";
import { useStoredFlag } from "./useStoredFlag";
import {
  COMPACT_VIEWPORT_QUERY,
  PHONE_PORTRAIT_QUERY,
  SHORT_LANDSCAPE_QUERY,
  useMediaQuery,
} from "./useMediaQuery";
import { Button } from "./Button";
import { Popup } from "./Popup";
import { Action } from "./Action";
import { Chat, type ReceivedChatMessage } from "./Chat";
import { CardPreview } from "./CardPreview";
import { MultiCardPreview } from "./MultiCardPreview";
import { CardPicturesProvider, type CardPictures } from "./cardImage";
import { CoinToss } from "./CoinToss";

const scrollbarStyles = `
  .custom-scrollbar::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }
  
  .custom-scrollbar::-webkit-scrollbar-track {
    background: rgba(0, 0, 0, 0.2);
    border-radius: 4px;
  }
  
  .custom-scrollbar::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.3);
    border-radius: 4px;
  }
  
  .custom-scrollbar::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.5);
  }
  
  .custom-scrollbar {
    scrollbar-width: thin;
    scrollbar-color: rgba(255, 255, 255, 0.3) rgba(0, 0, 0, 0.2);
  }
`;

export interface DuelProps {
  /** The game's websocket, such as wss://site.example/api/games/abc/socket. */
  socketUrl: string;
  /** Where the card pictures come from. */
  pictures: CardPictures;
  playmat?: string;
  /**
   * Whether the opponent's cards are flipped from their natural orientation
   * towards the player looking at the screen. Defaults to false, which orients
   * them towards the opponent the way they would face across a physical table.
   * The opponent's mana is exempt: it always faces the player looking at the
   * screen, mirroring our own mana facing the opponent.
   */
  flipOpponentCards?: boolean;
  resolveChatUser?: DuelChatUserResolver;
  renderChatUserTrigger?: DuelChatUserTriggerRenderer;
  /**
   * Current usernames whose user-authored chat messages should be hidden.
   * Existing history is retained and re-filtered when this list changes.
   */
  blockedChatUsers?: readonly string[];
  devTools?: {
    cards: { uid: string; name: string }[];
    activePlayer: "host" | "guest" | "spectator";
    onPlayerSwitch: (player: "host" | "guest" | "spectator") => void;
  };
  onNewTurn?: (myTurn: boolean) => void;
  onLeaveDuel?: () => void;
  onDuelFinished?: (message: DuelFinishedMessage) => void;
  /**
   * What the screen shows once the game has ended, such as the result and the
   * way back to the lobby. Without it the board counts down, then calls
   * onDuelFinished; with it, onDuelFinished is called at once.
   */
  renderDuelFinished?: (message: DuelFinishedMessage) => ReactNode;
}

interface DragState {
  virtualId: string;
  imageId: string;
  name?: string;
  sourceZone: DragZone;
  mouseX: number;
  mouseY: number;
  rotated?: boolean;
}


interface PreviewCard {
  name: string;
  imageId: string;
  /** The card on the board being viewed, whose statuses are shown with it. */
  virtualId?: string;
}

interface PreviewCards {
  text?: string;
  cards: PreviewCard[];
}

interface Action {}

/** If a closed card menu never reports that its card settled back into its
 * row, the card is shown again this long after closing anyway. */
const CARD_DROP_GRACE_MS = 1000;

/** How far a pointer may travel before a press counts as a drag rather than a
 * tap. A finger is much less precise than a mouse and always wobbles a little,
 * so touch needs more slack or every tap would be read as a drag. */
const MOUSE_DRAG_THRESHOLD = 5;
const TOUCH_DRAG_THRESHOLD = 12;


/** Leaving the duel, shown the same way on every screen size: a door with an
 * arrow heading out, in red because it forfeits the match. */
function ForfeitButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Forfeit"
      title="Forfeit"
      className="flex h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-md bg-gray-800 text-red-500 transition-colors hover:bg-gray-700 hover:text-red-400"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="h-5 w-5"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path
          fillRule="evenodd"
          d="M3 3a1 1 0 00-1 1v12a1 1 0 102 0V4a1 1 0 00-1-1zm10.293 9.293a1 1 0 001.414 1.414l3-3a1 1 0 000-1.414l-3-3a1 1 0 10-1.414 1.414L14.586 9H7a1 1 0 100 2h7.586l-1.293 1.293z"
          clipRule="evenodd"
        />
      </svg>
    </button>
  );
}

function ChatButton({
  onClick,
  className,
}: {
  onClick: () => void;
  className: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label="Open chat"
      className={`flex shrink-0 cursor-pointer items-center justify-center bg-gray-800 text-gray-200 transition-colors hover:bg-gray-700 ${className}`}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        className="h-5 w-5"
        viewBox="0 0 20 20"
        fill="currentColor"
      >
        <path
          fillRule="evenodd"
          d="M18 10c0 3.866-3.582 7-8 7a8.84 8.84 0 01-4.083-.98L2 17l1.338-3.123C2.493 12.767 2 11.434 2 10c0-3.866 3.582-7 8-7s8 3.134 8 7z"
          clipRule="evenodd"
        />
      </svg>
    </button>
  );
}

type ZoneKind = "mana" | "shields" | "battle" | "emptyBattle" | "hand";

/** How tall each kind of zone is. Most screens split the board's height by a
 * fixed share. An upright phone is far taller than it is wide, so there the
 * rows are capped by the board's width instead, which keeps several cards in
 * view across, and the height left over gathers around the phase bar rather
 * than as empty space inside every row. */
const ZONE_HEIGHTS: Record<"shares" | "upright", Record<ZoneKind, string>> = {
  shares: {
    mana: "h-[10%]",
    shields: "h-[10%]",
    battle: "h-[20%]",
    emptyBattle: "h-[20%]",
    hand: "h-[20%]",
  },
  upright: {
    mana: "min-h-0 flex-[1_1_0%] max-h-[14cqw]",
    shields: "min-h-0 flex-[1_1_0%] max-h-[20cqw]",
    battle: "min-h-0 flex-[1_1_0%] max-h-[30cqw]",
    emptyBattle: "min-h-0 flex-[1_1_0%] max-h-[18cqw]",
    hand: "min-h-0 flex-[1_1_0%] max-h-[30cqw]",
  },
};

/** The room the bottom bar, or the side rail, keeps clear of the board. */
const CONTROL_BAR_SIZE = "3.25rem";

function DevToolSection({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <fieldset className="border border-gray-700 rounded px-3 pb-3 pt-2">
      <legend className="px-2 text-xs text-gray-500">{title}</legend>
      {children}
    </fieldset>
  );
}

export function Duel({
  socketUrl,
  pictures,
  playmat,
  flipOpponentCards = false,
  resolveChatUser,
  renderChatUserTrigger,
  blockedChatUsers,
  devTools,
  onLeaveDuel,
  onDuelFinished,
  renderDuelFinished,
  onNewTurn,
}: DuelProps) {
  const [action, setAction] = useState<ActionMessage | null>(null);
  const [actionRevision, setActionRevision] = useState(0);
  const [actionError, setActionError] = useState<ActionWarningMessage | null>(
    null,
  );
  const [chatMessages, setChatMessages] = useState<ReceivedChatMessage[]>([]);
  const [wait, setWait] = useState("");
  const [warningMessage, setWarningMessage] = useState("");
  const [dots, setDots] = useState<"." | ".." | "...">(".");
  const [duelFinishedCountdown, setDuelFinishedCountdown] = useState<
    number | null
  >(null);
  const [duelFinishedRedirectFailed, setDuelFinishedRedirectFailed] =
    useState(false);
  const [confirmForfeit, setConfirmForfeit] = useState(false);
  const [confirmAttacks, setConfirmAttacks] = useStoredFlag(
    "duel-masters:confirm-attack-forfeit",
    true,
  );
  const [pendingAttack, setPendingAttack] = useState<{
    label: string;
    forfeit: AttackForfeit;
    run: () => void;
  } | null>(null);
  const [stopConfirmingAttacks, setStopConfirmingAttacks] = useState(false);

  const {
    connected,
    error,
    duelFinished,
    coinToss,
    sendTossCall,
    sendTurnChoice,
    send,
    sendJoinMatch,
    sendEndTurn,
    sendAddToBattlezone,
    sendAddToManazone,
    sendAttackPlayer,
    sendAttackCreature,
    sendTapAbility,
    sendResign,
    sendAction,
    sendChat,
    state,
    opponentDisconnected,
    reconnecting,
  } = useDuel({
    socketUrl,
    onActionMessage: (nextAction) => {
      setAction(nextAction);
      setActionError(null);
      setActionRevision((revision) => revision + 1);
    },
    onActionError: setActionError,
    onActionClose: () => {
      setAction(null);
      setActionError(null);
    },
    onChat: (data) => {
      setChatMessages((prev) => [...prev, { ...data, receivedAt: Date.now() }]);
    },
    onWarning: (data) => {
      setWarningMessage(data.message);
    },
    onWait: (data) => {
      setWait(data.message);
    },
    onEndWait: () => {
      setWait("");
    },
  });

  /** Answers a question on screen, naming its number. A preview that only
   * shows cards asks nothing, so it has no number and nothing is sent. */
  const answerQuestion = (question: ActionMessage, answer: ActionAnswer) => {
    if (question.prompt !== undefined) {
      sendAction(question.prompt, answer);
    }
  };

  // Whose turn it is only reaches the client as part of the match state, so a
  // new turn is the turn changing from the one previously seen rather than an
  // event of its own.
  const myTurn = state?.myTurn;
  const previousMyTurnRef = useRef<boolean | null>(null);

  useEffect(() => {
    if (myTurn === undefined) {
      return;
    }

    const previousMyTurn = previousMyTurnRef.current;
    previousMyTurnRef.current = myTurn;

    // The first state of the connection describes the turn already in progress,
    // whether the duel just started or the player reconnected mid-duel.
    if (previousMyTurn === null || previousMyTurn === myTurn) {
      return;
    }

    onNewTurn?.(myTurn);
  }, [myTurn, onNewTurn]);

  useEffect(() => {
    if (!duelFinished) {
      return;
    }

    setWarningMessage("");
    setWait("");
    setAction(null);
    setActionError(null);
    setConfirmForfeit(false);

    // The page shows its own result, so it hears of the end at once.
    if (renderDuelFinished) {
      onDuelFinished?.(duelFinished);
      return;
    }

    setDuelFinishedCountdown(5);
    setDuelFinishedRedirectFailed(false);

    let countdown = 5;
    const interval = window.setInterval(() => {
      countdown -= 1;

      if (countdown <= 0) {
        window.clearInterval(interval);
        setDuelFinishedCountdown(0);
        onDuelFinished?.(duelFinished);

        window.setTimeout(() => {
          setDuelFinishedRedirectFailed(true);
        }, 3000);
        return;
      }

      setDuelFinishedCountdown(countdown);
    }, 1000);

    return () => {
      window.clearInterval(interval);
    };
  }, [duelFinished, onDuelFinished, renderDuelFinished]);

  const duelFinishedWinner =
    duelFinished?.winner?.username ?? duelFinished?.winner?.uid ?? "No winner";

  const duelFinishedStatusText =
    duelFinishedCountdown !== null && duelFinishedCountdown > 0
      ? `Redirecting in ${duelFinishedCountdown}...`
      : "Redirecting...";

  const isCompact = useMediaQuery(COMPACT_VIEWPORT_QUERY);
  /** A phone on its side has height for little more than its cards, so the
   * controls go down the left edge instead of along the bottom. */
  const isShortLandscape = useMediaQuery(SHORT_LANDSCAPE_QUERY);
  const isUpright = useMediaQuery(PHONE_PORTRAIT_QUERY);
  const [rotateHintDismissed, setRotateHintDismissed] = useStoredFlag(
    "duel-masters:rotate-hint-dismissed",
    false,
  );
  const [chatOpen, setChatOpen] = useState(false);
  /** The decks, graveyards and hand counts take real width away from the board,
   * so a player who wants every pixel for their cards can fold them away. */
  const [infoPanelOpen, setInfoPanelOpen] = useState(true);

  // The drawer only exists on narrow screens; leaving it "open" while resizing
  // to desktop would otherwise strand the state.
  useEffect(() => {
    if (!isCompact) {
      setChatOpen(false);
    }
  }, [isCompact]);

  // A narrow screen has no width to spare for the decks and graveyards, so the
  // panel folds itself away on the way in. Going back to a wide screen leaves
  // it folded rather than second-guessing the player.
  useEffect(() => {
    if (isCompact) {
      setInfoPanelOpen(false);
    }
  }, [isCompact]);

  const [previewCard, setPreviewCard] = useState<PreviewCard | null>(null);
  const [multiCardView, setMultiCardView] = useState<{
    cards: { imageId: string; name: string }[];
    title: string;
  } | null>(null);

  /** The card whose action menu is open, and the zone it was opened in. */
  const [menuCard, setMenuCard] = useState<{
    virtualId: string;
    zone: BoardCard["zone"];
  } | null>(null);
  /** Cards whose menu has closed but which are still settling into their row. */
  const [droppingCardIds, setDroppingCardIds] = useState<readonly string[]>(
    [],
  );

  const liftedCardIds = useMemo(
    () =>
      menuCard ? [menuCard.virtualId, ...droppingCardIds] : droppingCardIds,
    [menuCard, droppingCardIds],
  );

  // A menu belongs to the card where the player opened it. If the card has
  // since left that zone, its actions no longer apply.
  const menuTarget =
    menuCard && state ? findBoardCard(state, menuCard.virtualId) : null;
  const menuIsCurrent = !!menuTarget && menuTarget.zone === menuCard?.zone;

  useEffect(() => {
    if (menuCard && !menuIsCurrent) {
      setDroppingCardIds((ids) => [...ids, menuCard.virtualId]);
      setMenuCard(null);
    }
  }, [menuCard, menuIsCurrent]);

  useEffect(() => {
    if (droppingCardIds.length === 0) {
      return;
    }

    const timeout = window.setTimeout(
      () => setDroppingCardIds([]),
      CARD_DROP_GRACE_MS,
    );
    return () => window.clearTimeout(timeout);
  }, [droppingCardIds]);

  useEffect(() => {
    const interval = setInterval(() => {
      setDots((prev) => {
        if (prev === ".") return "..";
        if (prev === "..") return "...";
        return ".";
      });
    }, 500);

    return () => clearInterval(interval);
  }, []);

  const [dragState, setDragState] = useState<DragState | null>(null);
  const [dropZone, setDropZone] = useState<DragZone | null>(null);
  const [dragStartPosition, setDragStartPosition] = useState<{
    x: number;
    y: number;
    virtualId: string;
    imageId: string;
    name?: string;
    sourceZone: DragZone;
    rotated: boolean;
    threshold: number;
  } | null>(null);

  const handleCardDragStart = (
    virtualId: string,
    imageId: string,
    name: string | undefined,
    sourceZone: DragZone,
    rotated: boolean,
    e: React.PointerEvent,
  ) => {
    // Ignore right clicks
    if (e.button === 2) {
      return;
    }

    // Record start position but don't start dragging yet
    setDragStartPosition({
      x: e.clientX,
      y: e.clientY,
      virtualId,
      imageId,
      name,
      sourceZone,
      rotated,
      // A finger is far less precise than a mouse, so it needs more slack
      // before a tap counts as a drag.
      threshold: e.pointerType === "mouse" ? MOUSE_DRAG_THRESHOLD : TOUCH_DRAG_THRESHOLD,
    });
  };

  const handleMouseMove = (e: PointerEvent) => {
    const clientX = e.clientX;
    const clientY = e.clientY;

    // Check if we should start dragging based on threshold
    if (dragStartPosition && !dragState) {
      const deltaX = Math.abs(clientX - dragStartPosition.x);
      const deltaY = Math.abs(clientY - dragStartPosition.y);

      if (
        deltaX > dragStartPosition.threshold ||
        deltaY > dragStartPosition.threshold
      ) {
        // Start dragging
        setDragState({
          virtualId: dragStartPosition.virtualId,
          imageId: dragStartPosition.imageId,
          name: dragStartPosition.name,
          sourceZone: dragStartPosition.sourceZone,
          mouseX: clientX,
          mouseY: clientY,
          rotated: dragStartPosition.rotated,
        });
        setDragStartPosition(null);
      }
      return;
    }

    if (!dragState) return;

    setDragState({
      ...dragState,
      mouseX: clientX,
      mouseY: clientY,
    });

    // Check all drop zones and find which one contains the cursor
    let foundZone: DragZone | null = null;
    const dropZones = document.querySelectorAll("[data-dropzone]");
    dropZones.forEach((zone) => {
      const rect = zone.getBoundingClientRect();
      if (
        clientX >= rect.left &&
        clientX <= rect.right &&
        clientY >= rect.top &&
        clientY <= rect.bottom
      ) {
        foundZone = zone.getAttribute("data-dropzone") as DragZone;
      }
    });

    setDropZone(foundZone);
  };

  /** Attacking or using a tap ability ends the summon step for the rest of
   * the turn. While that would still cost the player something, they are asked
   * first, unless they chose not to be. */
  const guardAttack = (label: string, run: () => void) => {
    const forfeit = confirmAttacks && state ? attackForfeit(state) : null;

    if (!forfeit) {
      run();
      return;
    }

    setStopConfirmingAttacks(false);
    setPendingAttack({ label, forfeit, run });
  };

  const myUsername = state?.me.username;
  const opponentUsername = state?.opponent.username;
  // Kept stable while the players stay the same, so the chat, which only
  // redraws when what it is given changes, does not redraw on every state.
  const chatPlayers = useMemo(
    () =>
      myUsername !== undefined && opponentUsername !== undefined
        ? { me: myUsername, opponent: opponentUsername }
        : undefined,
    [myUsername, opponentUsername],
  );
  const showChatCard = useCallback(
    (card: ChatCard) =>
      setPreviewCard({
        imageId: card.uid,
        name: card.name,
        virtualId: card.virtualId,
      }),
    [],
  );

  const showCardPreview = (card: CardState | ShieldState) =>
    setPreviewCard({
      imageId: card.uid,
      name: "name" in card ? card.name : "",
      virtualId: card.virtualId,
    });

  /** Opens the action menu of a card on the board. A card that can only be
   * looked at is enlarged straight away instead. */
  const openCardMenu = (virtualId: string) => {
    const target = state ? findBoardCard(state, virtualId) : null;

    if (!state || !target) {
      return;
    }

    const actions = cardActions(target, state);

    if (actions.length === 1 && actions[0].id === "view") {
      showCardPreview(target.card);
    } else if (actions.length > 0) {
      setMenuCard({ virtualId, zone: target.zone });
    }
  };

  const closeCardMenu = () => {
    if (menuCard) {
      setDroppingCardIds((ids) => [...ids, menuCard.virtualId]);
      setMenuCard(null);
    }
  };

  const runCardAction = (id: CardActionId, card: CardState | ShieldState) => {
    const { virtualId } = card;
    closeCardMenu();

    switch (id) {
      case "play":
        sendAddToBattlezone(virtualId);
        break;
      case "charge":
        sendAddToManazone(virtualId);
        break;
      case "attackPlayer":
        guardAttack("Attack", () => sendAttackPlayer(virtualId));
        break;
      case "attackCreature":
        guardAttack("Attack", () => sendAttackCreature(virtualId));
        break;
      case "tapAbility":
        guardAttack("Use tap ability", () => sendTapAbility(virtualId));
        break;
      case "view":
        showCardPreview(card);
        break;
      default: {
        const unhandled: never = id;
        throw new Error(`Unhandled card action ${unhandled}`);
      }
    }
  };

  const handleMouseUp = () => {
    // A press that never moved far enough to become a drag is a tap
    if (dragStartPosition && !dragState) {
      openCardMenu(dragStartPosition.virtualId);
      setDragStartPosition(null);
      return;
    }

    if (!dragState || !dropZone) {
      setDragState(null);
      setDropZone(null);
      setDragStartPosition(null);
      return;
    }

    // Handle drop actions based on source and target zones
    if (dragState.sourceZone === "hand" && dropZone === "myManazone") {
      sendAddToManazone(dragState.virtualId);
    } else if (dragState.sourceZone === "hand" && dropZone === "myPlayzone") {
      sendAddToBattlezone(dragState.virtualId);
    } else if (
      dragState.sourceZone === "myPlayzone" &&
      dropZone === "opponentPlayzone"
    ) {
      const { virtualId } = dragState;
      guardAttack("Attack", () => sendAttackCreature(virtualId));
    } else if (
      dragState.sourceZone === "myPlayzone" &&
      dropZone === "opponentShieldzone"
    ) {
      const { virtualId } = dragState;
      guardAttack("Attack", () => sendAttackPlayer(virtualId));
    }

    setDragState(null);
    setDropZone(null);
    setDragStartPosition(null);
  };

  const handlePointerCancel = () => {
    setDragState(null);
    setDropZone(null);
    setDragStartPosition(null);
  };

  useEffect(() => {
    if (dragState || dragStartPosition) {
      window.addEventListener("pointermove", handleMouseMove);
      window.addEventListener("pointerup", handleMouseUp);
      // A pointer can be taken away mid gesture, by the system or by a browser
      // scroll gesture winning. Without this the card would stay stuck to the
      // cursor with no way to drop it.
      window.addEventListener("pointercancel", handlePointerCancel);

      return () => {
        window.removeEventListener("pointermove", handleMouseMove);
        window.removeEventListener("pointerup", handleMouseUp);
        window.removeEventListener("pointercancel", handlePointerCancel);
      };
    }
  }, [dragState, dropZone, dragStartPosition]);

  useEffect(() => {
    if (connected) {
      sendJoinMatch();
    }
  }, [connected]);

  // The coin toss comes before the first state, so it shows over the waiting
  // screen as well as over the board.
  const coinTossOverlay = coinToss && (
    <CoinToss toss={coinToss} onCall={sendTossCall} onChooseTurn={sendTurnChoice} />
  );

  if (!state) {
    return (
      <div className="flex h-dvh items-center justify-center bg-[linear-gradient(45deg,rgb(29,33,42),rgb(20,16,21))] text-gray-300">
        <p role="status">Waiting for both players to join...</p>
        {coinTossOverlay}
      </div>
    );
  }

  const isSpectating = state.spectator;

  const getValidDropZones = (sourceZone: DragZone): DragZone[] => {
    if (sourceZone === "hand") {
      const zones: DragZone[] = ["myPlayzone", "myManazone"];
      return zones;
    }
    if (sourceZone === "myPlayzone") {
      return ["opponentPlayzone", "opponentShieldzone"];
    }
    return [];
  };

  const isValidDropZone = (zone: DragZone): boolean => {
    if (!dragState) return false;
    return getValidDropZones(dragState.sourceZone).includes(zone);
  };

  const getDropZoneColor = (zone: DragZone): DropHighlight => {
    if (!dragState || !isValidDropZone(zone)) return null;

    // Check if the action is actually allowed
    if (dragState.sourceZone === "hand") {
      // Find the card being dragged
      const draggedCard = state?.me.hand.find(
        (c) => c.virtualId === dragState.virtualId,
      );

      if (zone === "myPlayzone") {
        // Check if card can be played. Attacking or using a tap ability moves
        // the match into the attack step, after which nothing can be summoned.
        return draggedCard &&
          cardHasFlag(draggedCard.flags, PLAYABLE_FLAG) &&
          !state?.hasAttackedThisRound
          ? "green"
          : "red";
      }

      if (zone === "myManazone") {
        // Check if mana can be added
        return !state?.hasAddedManaThisRound &&
          !state?.hasAttackedThisRound &&
          state?.canChargeManaThisRound
          ? "green"
          : "red";
      }
    }

    // Check if attacking opponent's battlezone when it's empty
    if (dragState.sourceZone === "myPlayzone" && zone === "opponentPlayzone") {
      // If opponent has no creatures, highlight red
      return state?.opponent.playzone.length === 0 ? "red" : "green";
    }

    // For other zones (attacking shields), always green if valid
    return "green";
  };

  const showGraveyard = (cards: CardState[], title: string) =>
    setMultiCardView({
      cards: cards.map((card) => ({ imageId: card.uid, name: card.name })),
      title,
    });

  /** What every card on the board shares, whichever zone it is in. */
  const boardCardOptions = {
    dragState,
    onActivate: openCardMenu,
    onRightClick: (imageId: string, name?: string, virtualId?: string) =>
      setPreviewCard({ imageId, name: name || "", virtualId }),
  };

  /** Where a narrow screen puts the controls that would otherwise sit beside
   * the board: a bar along the bottom, within thumb reach, or a rail down the
   * left edge of a phone on its side. */
  const compactControls = !isCompact
    ? null
    : isShortLandscape
      ? "rail"
      : "bar";
  const zoneHeights = ZONE_HEIGHTS[isUpright ? "upright" : "shares"];
  const battleHeight = (cards: unknown[]) =>
    cards.length > 0 ? zoneHeights.battle : zoneHeights.emptyBattle;

  const boardClearance =
    compactControls && !isSpectating
      ? compactControls === "rail"
        ? { paddingLeft: CONTROL_BAR_SIZE }
        : { paddingBottom: CONTROL_BAR_SIZE }
      : undefined;

  return (
    <CardPicturesProvider value={pictures}>
      <style>{scrollbarStyles}</style>
      <BoardMotion
        state={state}
        liftedCardIds={liftedCardIds}
      >
        <div
          className="w-full h-screen text-white flex bg-[linear-gradient(45deg,rgb(29,33,42),rgb(20,16,21))] bg-cover bg-no-repeat gap-2 p-1 custom-scrollbar"
          style={{
            height: "100dvh",
            ...(playmat && {
              backgroundImage: `url(${JSON.stringify(playmat)}), linear-gradient(45deg, rgb(29, 33, 42), rgb(20, 16, 21))`,
              backgroundPosition: "center",
            }),
            ...(dragState && { cursor: "grabbing" }),
          }}
        >
          {/* On a narrow screen this column is replaced by a chat drawer and a
              bottom control bar, so the board can use the full width. */}
          <div
            className={
              isCompact
                ? "fixed inset-y-0 left-0 z-40 flex w-[86vw] max-w-[340px] flex-col gap-2 bg-[rgb(20,16,21)] p-2 shadow-2xl transition-transform duration-200" +
                  (chatOpen ? " translate-x-0" : " -translate-x-full")
                : "w-[300px] flex flex-col gap-2"
            }
          >
            {isCompact && (
              <div className="flex shrink-0 items-center justify-between">
                <span className="text-sm font-semibold">Chat</span>
                <button
                  type="button"
                  onClick={() => setChatOpen(false)}
                  aria-label="Close chat"
                  className="flex h-8 w-8 cursor-pointer items-center justify-center rounded-full bg-gray-800 text-gray-300 transition-colors hover:bg-gray-700"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="h-5 w-5"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                  >
                    <path
                      fillRule="evenodd"
                      d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                      clipRule="evenodd"
                    />
                  </svg>
                </button>
              </div>
            )}

            {/* Devtools */}
            {devTools && (
              <div className="bg-black/30 rounded-md overflow-hidden p-3 text-sm">
                <p className="mb-3 font-semibold">Development Tools</p>

                <DevToolSection title="Player Switch">
                  <div className="flex gap-2">
                    <div className="flex-1">
                      <Button
                        variant={
                          devTools.activePlayer === "host" ? "default" : "gray"
                        }
                        onClick={() => devTools.onPlayerSwitch("host")}
                      >
                        Host
                      </Button>
                    </div>
                    <div className="flex-1">
                      <Button
                        variant={
                          devTools.activePlayer === "guest" ? "default" : "gray"
                        }
                        onClick={() => devTools.onPlayerSwitch("guest")}
                      >
                        Guest
                      </Button>
                    </div>
                    <div className="flex-1">
                      <Button
                        variant={
                          devTools.activePlayer === "spectator"
                            ? "default"
                            : "gray"
                        }
                        onClick={() => devTools.onPlayerSwitch("spectator")}
                      >
                        Spectator
                      </Button>
                    </div>
                  </div>
                </DevToolSection>

                <div className="mt-3">
                  <DevToolSection title="Initialize">
                    <Button variant="gray" onClick={() => sendChat("/init all")}>
                      Initialize zones with 1 of each race
                    </Button>
                  </DevToolSection>
                </div>

                <div className="mt-3">
                  <DevToolSection title="Add Cards">
                    <div className="flex gap-2">
                      <div className="flex-1">
                        <select
                          className="w-full bg-gray-800 text-white pl-2 pr-8 py-[0.4rem] rounded border border-gray-700 focus:outline-none focus:border-blue-500 text-xs appearance-none bg-[url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2212%22%20height%3D%2212%22%20viewBox%3D%220%200%2012%2012%22%3E%3Cpath%20fill%3D%22%23ffffff%22%20d%3D%22M6%209L1%204h10z%22%2F%3E%3C%2Fsvg%3E')] bg-[length:12px_12px] bg-[position:right_0.5rem_center] bg-no-repeat"
                          id="card-selector"
                          defaultValue=""
                        >
                          <option value="" disabled>
                            Select a card...
                          </option>
                          {devTools?.cards.map((card) => (
                            <option key={card.uid} value={card.uid}>
                              {card.name}
                            </option>
                          ))}
                        </select>
                      </div>

                      <div className="">
                        <Button
                          variant="gray"
                          onClick={() => {
                            const select = document.getElementById(
                              "card-selector",
                            ) as HTMLSelectElement;
                            if (select.value) {
                              sendChat(`/add ${select.value}`);
                            }
                          }}
                        >
                          Add
                        </Button>
                      </div>
                    </div>
                  </DevToolSection>
                </div>
              </div>
            )}

            {/* Chat */}
            <div className="flex-1 bg-black/30 rounded-md overflow-hidden">
              <Chat
                messages={chatMessages}
                onSendMessage={sendChat}
                resolveUser={resolveChatUser}
                renderUserTrigger={renderChatUserTrigger}
                blockedUsers={blockedChatUsers}
                players={chatPlayers}
                onCardClick={showChatCard}
              />
            </div>
          </div>

          {/* Tapping the dimmed board closes the drawer */}
          {isCompact && chatOpen && (
            <div
              className="fixed inset-0 z-30 bg-black/60"
              onClick={() => setChatOpen(false)}
            />
          )}

          {/* min-w-0 lets the board give width to the info column instead of
              growing past it; the zones inside scroll horizontally. */}
          <div
            className={`flex min-w-0 flex-1 flex-col h-full ${
              isUpright ? "[container-type:inline-size]" : ""
            }`}
            style={boardClearance}
          >
            <BoardZone
              zone="opponentManazone"
              anchor={anchorId("opponent", "manazone")}
              heightClassName={zoneHeights.mana}
              dropHighlight={getDropZoneColor("opponentManazone")}
              rowClassName="pb-1"
            >
              {state.opponent.manazone.map(
                CreateCard({
                  ...boardCardOptions,
                  // Mana always faces the player looking at the screen
                  flipped: false,
                  muteWhenTapped: true,
                  zone: "opponentManazone",
                }),
              )}
            </BoardZone>
            <BoardZone
              zone="opponentShieldzone"
              anchor={anchorId("opponent", "shieldzone")}
              heightClassName={zoneHeights.shields}
              centered
              dropHighlight={getDropZoneColor("opponentShieldzone")}
            >
              {state.opponent.shieldzone.map(
                CreateCard({
                  ...boardCardOptions,
                  flipped: !flipOpponentCards,
                  shieldMap: state.opponent.shieldMap,
                  shield: true,
                  zone: "opponentShieldzone",
                }),
              )}
            </BoardZone>
            <BoardZone
              zone="opponentPlayzone"
              anchor={anchorId("opponent", "playzone")}
              heightClassName={battleHeight(state.opponent.playzone)}
              rowClassName={`px-1 pb-1 ${STATUS_LANE_CLASS}`}
              dropHighlight={getDropZoneColor("opponentPlayzone")}
            >
              {state.opponent.playzone.map(
                CreateCard({
                  ...boardCardOptions,
                  flipped: !flipOpponentCards,
                  powers: state.opponent.powers,
                  statuses: state.opponent.statuses,
                  attacker: state.attacker,
                  zone: "opponentPlayzone",
                }),
              )}
            </BoardZone>
            {isUpright && <div className="min-h-0 flex-[0.2_1_0%]" />}
            <PhaseBar
              state={state}
              onEndTurn={!isSpectating && !isCompact ? sendEndTurn : undefined}
              compact={isCompact}
              slim={isShortLandscape}
            />
            {isUpright && <div className="min-h-0 flex-[0.2_1_0%]" />}
            <BoardZone
              zone="myPlayzone"
              anchor={anchorId("me", "playzone")}
              heightClassName={battleHeight(state.me.playzone)}
              rowClassName={`px-1 pb-1 ${STATUS_LANE_CLASS}`}
              dropHighlight={getDropZoneColor("myPlayzone")}
            >
              {state.me.playzone.map(
                CreateCard({
                  ...boardCardOptions,
                  interactable: state?.myTurn,
                  powers: state.me.powers,
                  statuses: state.me.statuses,
                  attacker: state.attacker,
                  zone: "myPlayzone",
                  draggable: state.myTurn,
                  onDragStart: handleCardDragStart,
                }),
              )}
            </BoardZone>
            <BoardZone
              zone="myShieldzone"
              anchor={anchorId("me", "shieldzone")}
              heightClassName={zoneHeights.shields}
              centered
              dropHighlight={getDropZoneColor("myShieldzone")}
            >
              {state.me.shieldzone.map(
                CreateCard({
                  ...boardCardOptions,
                  shieldMap: state.me.shieldMap,
                  shield: true,
                  zone: "myShieldzone",
                }),
              )}
            </BoardZone>
            <BoardZone
              zone="myManazone"
              anchor={anchorId("me", "manazone")}
              heightClassName={zoneHeights.mana}
              dropHighlight={getDropZoneColor("myManazone")}
            >
              {state.me.manazone.map(
                CreateCard({
                  ...boardCardOptions,
                  flipped: true,
                  muteWhenTapped: true,
                  zone: "myManazone",
                }),
              )}
            </BoardZone>
            {isSpectating ? (
              <div
                className={`relative w-full ${zoneHeights.hand}`}
                data-dropzone="hand"
              >
                <div className="absolute inset-0 bg-black/30 rounded-md flex items-center justify-center">
                  <div className="flex flex-col items-center gap-3 text-center">
                    <span className="text-xs font-semibold uppercase tracking-widest text-gray-400">
                      Spectating
                    </span>
                    <div className="flex items-center gap-8">
                      <span className="text-sm font-semibold text-white">
                        {state.me.username}
                      </span>
                      <span className="text-gray-500 text-lg font-bold">vs</span>
                      <span className="text-sm font-semibold text-white">
                        {state.opponent.username}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            ) : (
              <BoardZone
                zone="hand"
                anchor={anchorId("me", "hand")}
                heightClassName={zoneHeights.hand}
                rowClassName="pt-1 p-px"
              >
                {state.me.hand.map(
                  CreateCard({
                    ...boardCardOptions,
                    interactable: state?.myTurn,
                    canAddToManazone:
                      !state.hasAddedManaThisRound &&
                      !state.hasAttackedThisRound &&
                      state.canChargeManaThisRound,
                    onAddToBattlezone: (virtualId) => {
                      sendAddToBattlezone(virtualId);
                    },
                    onAddToManazone: (virtualId) => {
                      sendAddToManazone(virtualId);
                    },
                    onTapAbility: (virtualId) => {
                      sendTapAbility(virtualId);
                    },
                    zone: "hand",
                    draggable: state.myTurn,
                    onDragStart: handleCardDragStart,
                  }),
                )}
              </BoardZone>
            )}
          </div>

          {/* The controls of a narrow screen. The board keeps room for them so
              they never cover the cards. */}
          {compactControls && !isSpectating && (
            <div
              className={`fixed z-30 flex items-center gap-2 bg-[rgb(20,16,21)]/95 ${
                compactControls === "rail"
                  ? "inset-y-0 left-0 flex-col border-r border-white/10 py-2"
                  : "inset-x-0 bottom-0 border-t border-white/10 px-2"
              }`}
              style={
                compactControls === "rail"
                  ? {
                      width: CONTROL_BAR_SIZE,
                      // Clear of a notch on the phone's left side.
                      paddingLeft: "env(safe-area-inset-left)",
                    }
                  : {
                      height: CONTROL_BAR_SIZE,
                      paddingBottom: "env(safe-area-inset-bottom)",
                    }
              }
            >
              <ChatButton
                onClick={() => setChatOpen(true)}
                className="h-9 w-9 rounded-md"
              />

              <div className="min-h-0 min-w-0 flex-1" />
              <div
                className={
                  compactControls === "rail" ? "w-11" : "w-[6.5rem] shrink-0"
                }
              >
                <Button
                  onClick={sendEndTurn}
                  disabled={!state.myTurn}
                  disabledTooltip="It's not your turn"
                  size={compactControls === "rail" ? "narrow" : "default"}
                >
                  End turn
                </Button>
              </div>

              {/* Forfeit sits here rather than in the top corner so every control
                  a player needs is along one edge, within thumb reach. */}
              <ForfeitButton onClick={() => setConfirmForfeit(true)} />
            </div>
          )}

          {isUpright && !rotateHintDismissed && (
            <div className="fixed inset-x-2 top-2 z-30 flex items-center gap-3 rounded-lg bg-gray-900/95 px-3 py-2 text-xs text-gray-200 shadow-lg">
              <span className="min-w-0 flex-1">
                Turn your phone sideways for a bigger board.
              </span>
              <button
                type="button"
                onClick={() => setRotateHintDismissed(true)}
                className="shrink-0 cursor-pointer rounded-md bg-gray-700 px-2.5 py-1 font-medium text-white hover:bg-gray-600"
              >
                Got it
              </button>
            </div>
          )}

          {/* Spectators get the chat toggle without the control bar */}
          {isCompact && isSpectating && (
            <ChatButton
              onClick={() => setChatOpen(true)}
              className="fixed bottom-2 left-2 z-30 h-10 w-10 rounded-full shadow-lg"
            />
          )}

          {/* Forfeit - Top Right. On a narrow screen it lives in the bottom bar
              instead, beside End turn. */}
          {!isSpectating && !isCompact && (
            <div className="fixed right-[0.5vw] top-[0.5vh] z-30">
              <ForfeitButton onClick={() => setConfirmForfeit(true)} />
            </div>
          )}

          {/* Player Info Panel - Right Side. It occupies real width beside the
              board rather than floating over it, so a full mana or battle zone
              scrolls behind the panel's edge instead of disappearing under it. */}
          {/* The negative margins cancel the board container's padding so the
              panel runs from the top of the screen to the bottom and sits flush
              against the right edge. */}
          {/* On a narrow screen the panel is a drawer over the board, which
              would otherwise have to give up width it does not have. */}
          {isCompact && infoPanelOpen && (
            <div
              className="fixed inset-0 z-[39] bg-black/50"
              onClick={() => setInfoPanelOpen(false)}
            />
          )}
          <div
            className={
              isCompact
                ? `fixed inset-y-0 right-0 z-40 flex items-stretch bg-[rgb(20,16,21)] shadow-2xl transition-transform duration-200 ${
                    infoPanelOpen ? "translate-x-0" : "translate-x-full"
                  }`
                : "relative -my-1 -mr-1 flex shrink-0 items-stretch"
            }
          >
            {/* Collapse toggle: a half circle budding out of the panel's left
                edge, sharing its background so the two read as one shape. */}
            <button
              type="button"
              onClick={() => setInfoPanelOpen((open) => !open)}
              aria-expanded={infoPanelOpen}
              aria-label={
                infoPanelOpen ? "Collapse player info" : "Expand player info"
              }
              title={
                infoPanelOpen
                  ? "Hide decks and graveyards"
                  : "Show decks and graveyards"
              }
              /* The bump sits entirely outside the panel rather than overlapping
                 it: the panel's background is translucent, so any overlap would
                 stack into a darker patch instead of matching it. */
              className={`absolute left-0 top-1/2 z-30 flex -translate-x-full -translate-y-1/2 cursor-pointer items-center justify-center rounded-l-full pl-1 text-gray-300 focus:outline-none ${
                // A finger needs a bigger tab, and the drawer a solid one.
                isCompact
                  ? "h-16 w-8 bg-[rgb(20,16,21)]"
                  : "h-10 w-5 bg-black/30"
              }`}
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                className={`h-5 w-5 ${infoPanelOpen ? "" : "rotate-180"}`}
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fillRule="evenodd"
                  d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
                  clipRule="evenodd"
                />
              </svg>
            </button>

            {/* Collapsed, a sliver of the panel stays put: just enough to show
                it is still there and to give the toggle an edge to bud from
                rather than floating over the board. */}
            {!infoPanelOpen && !isCompact && (
              <div className="w-2 shrink-0 bg-black/30" />
            )}

            {/* A drawer keeps its contents while closed, so it can slide. */}
            {(infoPanelOpen || isCompact) && (
              /* Wide enough for the longest label, "Graveyard [10]", to stay on
                 one line: that is about 6.6em, and the labels are 1.2vh, so the
                 text needs ~8vh. The pixel bounds match where the label's own
                 clamp() bottoms out and tops off, plus the 1rem of padding. */
              <div
                className="flex w-[calc(8.8vh+1rem)] min-w-[90px] max-w-[116px] shrink-0 flex-col overflow-y-auto overflow-x-hidden bg-black/30 px-2 custom-scrollbar"
                {...CLIPS_CARDS_PROPS}
              >
                {/* m-auto rather than justify-center so the top stays reachable
                    when the two sections are taller than the screen */}
                <div className="m-auto flex w-full flex-col gap-[0.5vh] py-1 md:gap-[5vh]">
                  {/* Opponent Section */}
                  <div className="rounded-lg flex flex-col gap-[1vh]">
                    <HandCount side="opponent" count={state.opponent.handCount} />
                    <DeckPile side="opponent" count={state.opponent.deck} />
                    <GraveyardPile
                      side="opponent"
                      cards={state.opponent.graveyard}
                      onOpen={() =>
                        showGraveyard(
                          state.opponent.graveyard,
                          "Opponent's Graveyard",
                        )
                      }
                    />
                  </div>

                  {/* Player Section */}
                  <div className="rounded-lg flex flex-col gap-[1vh]">
                    {/* A player sees their own hand on the board instead */}
                    {isSpectating && (
                      <HandCount side="me" count={state.me.handCount} />
                    )}
                    <GraveyardPile
                      side="me"
                      cards={state.me.graveyard}
                      onOpen={() =>
                        showGraveyard(state.me.graveyard, "My Graveyard")
                      }
                    />
                    <DeckPile side="me" count={state.me.deck} />
                  </div>
                </div>
              </div>
            )}
          </div>

          <PhaseAnnouncer state={state} />

          <AnimatePresence>
            {menuCard && menuIsCurrent && menuTarget && (
              <CardMenu
                key={menuCard.virtualId}
                virtualId={menuCard.virtualId}
                imageId={menuTarget.card.uid}
                name={"name" in menuTarget.card ? menuTarget.card.name : undefined}
                actions={cardActions(menuTarget, state)}
                layout={isCompact ? "sheet" : "overlay"}
                onAction={(id) => runCardAction(id, menuTarget.card)}
                onClose={closeCardMenu}
                onDropped={() =>
                  setDroppingCardIds((ids) =>
                    ids.filter((id) => id !== menuCard.virtualId),
                  )
                }
              />
            )}
          </AnimatePresence>

          {/* Floating card that follows cursor */}
          {dragState && (
            <div
              className="fixed pointer-events-none z-50"
              style={{
                left: dragState.mouseX,
                top: dragState.mouseY,
                transform: "translate(-50%, -50%)",
                pointerEvents: "none",
              }}
            >
              <img
                src={pictures.card(dragState.imageId)}
                alt={dragState.name || "Card"}
                className={`h-[150px] opacity-90 ${
                  dragState.rotated ? "rotate-90" : ""
                }`}
                style={{ pointerEvents: "none", borderRadius: "5%" }}
              />
            </div>
          )}
        </div>
      </BoardMotion>

      <Popup
        visible={!!duelFinished}
        title="Duel Finished"
        maxWidth="500px"
        closeOnOutsideClick={false}
        showCloseButton={false}
      >
        {duelFinished && renderDuelFinished ? (
          renderDuelFinished(duelFinished)
        ) : (
          <div className="p-6 text-white">
            <p className="text-lg font-semibold">
              {duelFinishedWinner} won the duel
            </p>
            <p className="mt-3 text-sm text-gray-400">{duelFinishedStatusText}</p>
            {duelFinishedRedirectFailed && (
              <p className="mt-3 text-sm text-gray-400">
                Failed to redirect, please close the page manually
              </p>
            )}
          </div>
        )}
      </Popup>

      <Popup
        visible={confirmForfeit}
        onClose={() => setConfirmForfeit(false)}
        title="Forfeit"
        maxWidth="500px"
        closeOnOutsideClick={true}
      >
        <div className="p-6 text-white">
          <p>
            Are you sure you want to forfeit? Your opponent wins the duel
            immediately.
          </p>
          <div className="flex gap-3 mt-6">
            <Button variant="gray" onClick={() => setConfirmForfeit(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={() => {
                setConfirmForfeit(false);
                sendResign();
              }}
            >
              Forfeit
            </Button>
          </div>
        </div>
      </Popup>

      {pendingAttack && (
        <Popup
          visible={true}
          onClose={() => setPendingAttack(null)}
          title="End your summon step?"
          maxWidth="500px"
          closeOnOutsideClick={true}
        >
          <div className="p-6 text-white">
            <p>{describeAttackForfeit(pendingAttack.forfeit)}</p>
            <label className="mt-4 flex cursor-pointer items-center gap-2 text-sm text-gray-400">
              <input
                type="checkbox"
                checked={stopConfirmingAttacks}
                onChange={(e) => setStopConfirmingAttacks(e.target.checked)}
              />
              Don't ask me again
            </label>
            <div className="flex gap-3 mt-6">
              <Button variant="gray" onClick={() => setPendingAttack(null)}>
                Keep playing
              </Button>
              <Button
                onClick={() => {
                  if (stopConfirmingAttacks) {
                    setConfirmAttacks(false);
                  }
                  setPendingAttack(null);
                  pendingAttack.run();
                }}
              >
                {pendingAttack.label}
              </Button>
            </div>
          </div>
        </Popup>
      )}

      <Popup
        visible={reconnecting}
        title="Disconnected"
        maxWidth="500px"
        closeOnOutsideClick={false}
        showCloseButton={false}
      >
        <div className="p-6 text-white ">
          <p>Attempting to reconnect{dots}</p>
          <div className="flex mt-6">
            <Button variant="destructive" onClick={onLeaveDuel}>
              Leave Duel
            </Button>
          </div>
        </div>
      </Popup>

      <Popup
        visible={opponentDisconnected}
        title="Opponent Disconnected"
        maxWidth="500px"
        closeOnOutsideClick={false}
        showCloseButton={false}
      >
        <div className="p-6 text-white">
          <p>
            Your opponent disconnected or left the match. Waiting for them to
            reconnect{dots}
          </p>
          <div className="flex mt-6">
            <Button variant="destructive" onClick={onLeaveDuel}>
              Leave Duel
            </Button>
          </div>
        </div>
      </Popup>

      <CardPreview
        visible={!!previewCard}
        imageId={previewCard?.imageId || null}
        name={previewCard?.name || null}
        power={
          state && previewCard?.virtualId
            ? findCreaturePower(state, previewCard.virtualId)
            : null
        }
        statuses={
          state && previewCard?.virtualId
            ? statusBadges(findCreatureStatuses(state, previewCard.virtualId))
            : undefined
        }
        onClose={() => setPreviewCard(null)}
      />

      <MultiCardPreview
        visible={!!multiCardView}
        cards={multiCardView?.cards || []}
        title={multiCardView?.title || ""}
        onClose={() => setMultiCardView(null)}
        onCardClick={(imageId, name) => {
          setPreviewCard({ imageId, name });
          setMultiCardView(null);
        }}
      />

      <Popup
        visible={!!warningMessage.length}
        onClose={() => setWarningMessage("")}
        title="Warning"
        maxWidth="500px"
        closeOnOutsideClick={false}
      >
        <div className="p-6 text-white ">{warningMessage}</div>
      </Popup>

      <Popup
        visible={!!wait.length}
        onClose={() => setWait("")}
        title="Wait"
        maxWidth="500px"
        closeOnOutsideClick={false}
        showCloseButton={false}
      >
        <div className="p-6 text-white ">
          {wait}
          {dots}
        </div>
      </Popup>

      {action && (
        <Action
          key={actionRevision}
          title={action.showCards ? "Card Preview" : "Action Required"}
          visible={true}
          error={actionError ? actionError.message : undefined}
          actionType={action.actionType}
          cards={Array.isArray(action.cards) ? action.cards : undefined}
          cardsObject={
            typeof action.cards === "object" && !Array.isArray(action.cards)
              ? action.cards
              : undefined
          }
          // Either player's shields can end up in a prompt, and a virtualId
          // belongs to one card, so the two maps merge without colliding.
          shieldMap={{ ...state.opponent.shieldMap, ...state.me.shieldMap }}
          showCards={action.showCards}
          text={action.text}
          minSelections={action.minSelections || 0}
          maxSelections={action.maxSelections || 0}
          cancellable={action.cancellable || false}
          unselectableCards={action.unselectableCards}
          choices={action.choices || null}
          onChoose={(answer) => answerQuestion(action, answer)}
          onClose={() => answerQuestion(action, { cards: [], cancel: true })}
          onDismiss={() => setAction(null)}
          onCardRightClick={(imageId, name) =>
            setPreviewCard({ imageId, name: name || "" })
          }
        ></Action>
      )}
      {coinTossOverlay}
    </CardPicturesProvider>
  );
}

function CreateCard(
  options: {
    interactable?: boolean;
    canAddToManazone?: boolean;
    flipped?: boolean;
    /** virtualId -> shield number, as sent by the server in the match state. */
    shieldMap?: Record<string, number>;
    selected?: (virtualId: string) => boolean;
    onAddToBattlezone?: (virtualId: string) => void;
    onAddToManazone?: (virtualId: string) => void;
    onTapAbility?: (virtualId: string) => void;
    dragState?: DragState | null;
    zone?: DragZone;
    draggable?: boolean;
    onDragStart?: (
      virtualId: string,
      imageId: string,
      name: string | undefined,
      sourceZone: DragZone,
      rotated: boolean,
      e: React.PointerEvent,
    ) => void;
    onRightClick?: (
      imageId: string,
      name?: string,
      virtualId?: string,
    ) => void;
    onActivate?: (virtualId: string) => void;
    /** Grey out tapped cards, which in a mana zone have been used up. */
    muteWhenTapped?: boolean;
    shield?: boolean;
    /** Creatures' power, by virtualId. */
    powers?: Record<string, CreaturePower>;
    /** Creatures' abilities and limits, by virtualId. */
    statuses?: Record<string, CreatureStatus[]>;
    /** The creature attacking right now, by virtualId. */
    attacker?: string;
  } = {},
) {
  return (card: CardState | ShieldState) => {
    const name = "name" in card && card.name ? card.name : undefined;
    const rotated = cardHasFlag(card.flags, TAPPED_FLAG);
    const isDragging = options.dragState?.virtualId === card.virtualId;

    return (
      <Card
        virtualId={card.virtualId}
        name={name}
        imageId={card.uid}
        // The server keeps a card's id for the whole duel, whatever zone it is
        // in. Keying by position instead would hand a card's element to its
        // neighbour whenever a card ahead of it left the row.
        key={card.virtualId}
        rotated={rotated}
        muted={rotated && !!options.muteWhenTapped}
        shield={options.shield}
        power={options.powers?.[card.virtualId]}
        summoningSick={cardHasFlag(card.flags, SUMMONING_SICK_FLAG)}
        statuses={options.statuses?.[card.virtualId]}
        attacking={
          !!options.attacker && options.attacker === card.virtualId
        }
        number={options.shieldMap?.[card.virtualId]}
        selected={options.selected ? options.selected(card.virtualId) : false}
        interactable={options.interactable}
        canAddToBattlezone={cardHasFlag(card.flags, PLAYABLE_FLAG)}
        canAddToManazone={options.canAddToManazone}
        onAddToBattlezone={options.onAddToBattlezone}
        onAddToManazone={options.onAddToManazone}
        onTapAbility={options.onTapAbility}
        flipped={options.flipped}
        isDragging={isDragging}
        draggable={options.draggable}
        onDragStart={(e) => {
          if (options.onDragStart && options.zone && card.uid) {
            options.onDragStart(
              card.virtualId,
              card.uid,
              name,
              options.zone,
              rotated,
              e,
            );
          }
        }}
        onRightClick={() => {
          if (options.onRightClick && card.uid) {
            options.onRightClick(card.uid, name, card.virtualId);
          }
        }}
        onActivate={
          options.onActivate
            ? () => options.onActivate?.(card.virtualId)
            : undefined
        }
      ></Card>
    );
  };
}
