import {
  Component,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { MotionConfig, motion } from "motion/react";
import { diffBoard } from "./boardDiff";
import {
  CARD_IMAGE_PLACEHOLDER,
  cardImageUrl,
  MUTED_CARD_FILTER,
  PLAIN_CARD_FILTER,
  SHIELD_TINT,
} from "./cardImage";
import { shatter } from "./shatter";
import { planEchoes, type Echo } from "./echoes";
import {
  CardPositionRegistry,
  type BoardMeasurements,
  type MeasuredCard,
  type ScreenBox,
} from "./cardPositions";
import {
  FADED_SCALE,
  flightDuration,
  holdTurnDraws,
  planFlights,
  rotationDelta,
  type BoardChange,
  type Flight,
} from "./flights";
import { bannerSeconds } from "./turnPhases";
import {
  formatPowerDelta,
  powerChanges,
  powerPopupPoint,
} from "./creaturePower";
import type { MatchState } from "./types";

const NO_CARDS: readonly string[] = [];

const RegistryContext = createContext<CardPositionRegistry | null>(null);
const HiddenCardsContext = createContext<ReadonlySet<string>>(new Set());

/** Standard ease in and out: the card lifts off gently and settles softly. */
const FLIGHT_EASE = [0.4, 0, 0.2, 1] as const;

/** If a flight's end is somehow never reported, its card is revealed anyway
 * this long after the flight should have landed, rather than staying hidden. */
const LANDING_GRACE_MS = 500;

/** Lets the board follow its cards' positions. Each card element registers
 * itself under its virtualId, so the board can be measured just before and
 * just after every update. Returns the ref to attach to the card element. */
export function useTrackedCard<T extends HTMLElement>(
  virtualId: string | undefined,
  imageId: string,
  rotation: number,
  muted = false,
) {
  const registry = useContext(RegistryContext);
  const ref = useRef<T>(null);

  useLayoutEffect(() => {
    const element = ref.current;

    if (!registry || !virtualId || !element) {
      return;
    }

    registry.trackCard(virtualId, { element, imageId, rotation, muted });
    return () => registry.untrackCard(virtualId, element);
  }, [registry, virtualId, imageId, rotation, muted]);

  return ref;
}

/** Registers a place cards fly to or from when their own element is not
 * there, such as a deck pile. See anchorId for how places are named. */
export function useBoardAnchor<T extends HTMLElement>(anchorId: string) {
  const registry = useContext(RegistryContext);
  const ref = useRef<T>(null);

  useLayoutEffect(() => {
    const element = ref.current;

    if (!registry || !element) {
      return;
    }

    registry.trackAnchor(anchorId, element);
    return () => registry.untrackAnchor(anchorId, element);
  }, [registry, anchorId]);

  return ref;
}

/** Whether a card is hidden because a copy of it is drawn above the board:
 * flying to where the card now is, or lifted out for its action menu. */
export function useIsCardHidden(virtualId: string | undefined): boolean {
  const hiddenCards = useContext(HiddenCardsContext);
  return !!virtualId && hiddenCards.has(virtualId);
}

/** Reads where a card on the board is drawn right now. */
export function useMeasureCard(): (virtualId: string) => MeasuredCard | null {
  const registry = useContext(RegistryContext);

  return useCallback(
    (virtualId: string) => registry?.measureCard(virtualId) ?? null,
    [registry],
  );
}

interface BoardSnapshotProps {
  state: MatchState;
  registry: CardPositionRegistry;
  onChange: (change: BoardChange) => void;
  children: ReactNode;
}

/** Measures the board on both sides of each match state update. It is a class
 * because only getSnapshotBeforeUpdate runs after React has worked out the
 * update but before it has changed the page, which is the one moment the old
 * positions can still be read. Wrapping the board means its componentDidUpdate
 * runs after every card inside has registered its new element. */
class BoardSnapshot extends Component<BoardSnapshotProps> {
  getSnapshotBeforeUpdate(
    previous: BoardSnapshotProps,
  ): BoardMeasurements | null {
    return previous.state !== this.props.state
      ? this.props.registry.measure()
      : null;
  }

  componentDidUpdate(
    previous: BoardSnapshotProps,
    _previousState: unknown,
    before: BoardMeasurements | null,
  ) {
    if (!before) {
      return;
    }

    const { state, registry, onChange } = this.props;

    onChange({
      previous: previous.state,
      next: state,
      moves: diffBoard(previous.state, state),
      before,
      after: registry.measure(),
    });
  }

  render() {
    return this.props.children;
  }
}

/** A change of power announced above a creature. */
interface PowerPopup {
  key: number;
  x: number;
  y: number;
  delta: number;
}

/** A broken shield bursting where it stood. */
interface Shattering {
  key: number;
  virtualId: string;
  imageId: string;
  box: ScreenBox;
}

/** How long the copy of a card that acted takes to grow and fade. */
const ECHO_SECONDS = 0.5;

/** How much the copy of a card that acted grows while it fades. */
const ECHO_SCALE = 1.3;

/** How solid the copy is when it appears. */
const ECHO_OPACITY = 0.85;

interface ActiveEcho extends Echo {
  key: number;
}

/** How long a shield takes to burst. Its card flies on to the hand meanwhile. */
const SHATTER_SECONDS = 0.6;

/** How long a power change stays on screen while it rises and fades. */
const POWER_POPUP_SECONDS = 1.4;

interface ActiveFlight extends Flight {
  /** Identifies this flight among later ones of the same card, so a card that
   * moves again mid-flight starts a fresh animation. */
  key: number;
}

/** Animates cards moving around the board. Every card that changes place
 * between two match states flies there on a layer above the board, since the
 * zones clip anything outside them. The game itself never waits for this: the
 * new state is on the board at once, and each card is only hidden until its
 * copy lands on it. */
export function BoardMotion({
  state,
  liftedCardIds = NO_CARDS,
  children,
}: {
  state: MatchState;
  /** Cards drawn above the board by their action menu, hidden in their row. */
  liftedCardIds?: readonly string[];
  children: ReactNode;
}) {
  const [registry] = useState(() => new CardPositionRegistry());
  const [flights, setFlights] = useState<ActiveFlight[]>([]);
  const [powerPopups, setPowerPopups] = useState<PowerPopup[]>([]);
  const [shatterings, setShatterings] = useState<Shattering[]>([]);
  const [echoes, setEchoes] = useState<ActiveEcho[]>([]);
  const nextKey = useRef(0);

  const handleChange = useCallback((change: BoardChange) => {
    const popups = powerChanges(change.previous, change.next).flatMap(
      ({ virtualId, delta }) => {
        const card = change.after.cards.get(virtualId);
        return card?.visible
          ? [{ key: nextKey.current++, delta, ...powerPopupPoint(card) }]
          : [];
      },
    );

    if (popups.length > 0) {
      setPowerPopups((current) => [...current, ...popups]);
    }

    // A shield bursts where it was seen last, before it left for the hand.
    const shattered = (change.next.brokenShields ?? []).flatMap((virtualId) => {
      const shield = change.before.cards.get(virtualId);
      return shield?.visible
        ? [
            {
              key: nextKey.current++,
              virtualId,
              imageId: shield.imageId,
              box: shield,
            },
          ]
        : [];
    });

    if (shattered.length > 0) {
      setShatterings((current) => [...current, ...shattered]);
    }

    // A drawn card leaves the deck as the draw phase is announced, after the
    // title announcing the turn.
    const planned = holdTurnDraws(
      planFlights(change),
      change,
      bannerSeconds({ kind: "turn", title: "", tone: "neutral" }),
    );
    const echoed = planEchoes(change, planned);

    if (echoed.length > 0) {
      setEchoes((current) => [
        ...current,
        ...echoed.map((echo) => ({ ...echo, key: nextKey.current++ })),
      ]);
    }

    if (planned.length === 0) {
      return;
    }

    // A card that moves again replaces its earlier flight. The others carry
    // on, so a quick follow-up update does not cut every animation short.
    const replaced = new Set(planned.map((flight) => flight.virtualId));

    setFlights((current) => [
      ...current.filter((flight) => !replaced.has(flight.virtualId)),
      ...planned.map((flight) => ({ ...flight, key: nextKey.current++ })),
    ]);
  }, []);

  const handleEchoed = useCallback((key: number) => {
    setEchoes((current) => current.filter((echo) => echo.key !== key));
  }, []);

  const handleShattered = useCallback((key: number) => {
    setShatterings((current) => current.filter((item) => item.key !== key));
  }, []);

  const handlePopupDone = useCallback((key: number) => {
    setPowerPopups((current) => current.filter((popup) => popup.key !== key));
  }, []);

  const handleLanded = useCallback((key: number) => {
    setFlights((current) => current.filter((flight) => flight.key !== key));
  }, []);

  const hiddenCards = useMemo(() => {
    const hidden = new Set(
      flights
        .filter((flight) => flight.landsOnCard)
        .map((flight) => flight.virtualId),
    );

    for (const virtualId of liftedCardIds) {
      hidden.add(virtualId);
    }

    return hidden;
  }, [flights, liftedCardIds]);

  return (
    // Players who ask their system for less motion get cards that appear in
    // place instead of flying.
    <MotionConfig reducedMotion="user">
      <RegistryContext.Provider value={registry}>
        <HiddenCardsContext.Provider value={hiddenCards}>
          <BoardSnapshot
            state={state}
            registry={registry}
            onChange={handleChange}
          >
            {children}
          </BoardSnapshot>
          <div className="pointer-events-none fixed inset-0 z-[25] overflow-hidden">
            {flights.map((flight) => (
              <FlyingCard
                key={flight.key}
                flight={flight}
                onLanded={handleLanded}
              />
            ))}
            {powerPopups.map((popup) => (
              <PowerChangePopup
                key={popup.key}
                popup={popup}
                onDone={handlePopupDone}
              />
            ))}
            {echoes.map((echo) => (
              <CardEcho key={echo.key} echo={echo} onDone={handleEchoed} />
            ))}
            {/* Drawn last, so the card flying out is seen through the shards */}
            {shatterings.map((shattering) => (
              <ShieldShatter
                key={shattering.key}
                shattering={shattering}
                onDone={handleShattered}
              />
            ))}
          </div>
        </HiddenCardsContext.Provider>
      </RegistryContext.Provider>
    </MotionConfig>
  );
}

function FlyingCard({
  flight,
  onLanded,
}: {
  flight: ActiveFlight;
  onLanded: (key: number) => void;
}) {
  const { from, to, fade, landsOnCard, muted, key, delay } = flight;
  // Only copies of greyed out cards animate the filter, which is costlier than
  // the transforms every copy uses.
  const greyed = muted.from || muted.to;
  const duration = flightDuration(flight);

  useEffect(() => {
    const timeout = window.setTimeout(
      () => onLanded(key),
      (delay + duration) * 1000 + LANDING_GRACE_MS,
    );
    return () => window.clearTimeout(timeout);
  }, [key, delay, duration, onLanded]);

  // The copy sits where the card ends up and is moved back to where it
  // started, so it only ever animates transforms and opacity, which the
  // browser can do without laying the page out again.
  return (
    <motion.img
      src={cardImageUrl(flight.imageId)}
      alt=""
      draggable={false}
      className="absolute shadow-lg"
      style={{
        left: to.x - to.width / 2,
        top: to.y - to.height / 2,
        width: to.width,
        height: to.height,
        borderRadius: "5%",
        // A card just drawn or revealed may not have loaded yet.
        ...CARD_IMAGE_PLACEHOLDER,
      }}
      initial={{
        x: from.x - to.x,
        y: from.y - to.y,
        scale: fade === "in" ? FADED_SCALE : from.height / to.height,
        rotate: to.rotation - rotationDelta(from.rotation, to.rotation),
        opacity: fade === "in" ? 0 : 1,
        ...(greyed && {
          filter: muted.from ? MUTED_CARD_FILTER : PLAIN_CARD_FILTER,
        }),
      }}
      animate={{
        x: 0,
        y: 0,
        scale: fade === "out" ? FADED_SCALE : 1,
        rotate: to.rotation,
        // A copy headed for a pile or out of sight melts away as it arrives,
        // since there is no card underneath for it to land on.
        opacity: fade === "out" ? 0 : landsOnCard ? 1 : [1, 1, 0],
        ...(greyed && {
          filter: muted.to ? MUTED_CARD_FILTER : PLAIN_CARD_FILTER,
        }),
      }}
      transition={{
        duration,
        delay,
        ease: FLIGHT_EASE,
        opacity:
          fade || landsOnCard
            ? { duration, delay, ease: FLIGHT_EASE }
            : { duration, delay, times: [0, 0.7, 1] },
      }}
      onAnimationComplete={() => onLanded(key)}
    />
  );
}

/** Announces a change of power by rising from the creature and fading. */
function PowerChangePopup({
  popup,
  onDone,
}: {
  popup: PowerPopup;
  onDone: (key: number) => void;
}) {
  const { key, x, y, delta } = popup;

  // Removed on a timer as well, so a popup can never linger on screen.
  useEffect(() => {
    const timeout = window.setTimeout(
      () => onDone(key),
      POWER_POPUP_SECONDS * 1000 + LANDING_GRACE_MS,
    );
    return () => window.clearTimeout(timeout);
  }, [key, onDone]);

  return (
    <motion.div
      aria-hidden="true"
      className={`absolute whitespace-nowrap text-xl font-extrabold leading-none drop-shadow-[0_2px_3px_rgba(0,0,0,0.9)] ${
        delta > 0 ? "text-green-400" : "text-red-400"
      }`}
      style={{ left: x, top: y, x: "-50%" }}
      initial={{ y: "-50%", opacity: 0, scale: 0.8 }}
      animate={{ y: "-250%", opacity: [0, 1, 1, 0], scale: 1 }}
      transition={{
        duration: POWER_POPUP_SECONDS,
        ease: "easeOut",
        opacity: { duration: POWER_POPUP_SECONDS, times: [0, 0.15, 0.7, 1] },
      }}
      onAnimationComplete={() => onDone(key)}
    >
      {formatPowerDelta(delta)}
    </motion.div>
  );
}

/** A broken shield bursting into pieces of blue glass from a point of
 * impact, with a flash where it was struck. */
function ShieldShatter({
  shattering,
  onDone,
}: {
  shattering: Shattering;
  onDone: (key: number) => void;
}) {
  const { key, virtualId, imageId, box } = shattering;
  const [{ impact, shards }] = useState(() => shatter(virtualId));

  // Removed on a timer as well, so the pieces can never linger on screen.
  useEffect(() => {
    const timeout = window.setTimeout(
      () => onDone(key),
      SHATTER_SECONDS * 1000 + LANDING_GRACE_MS,
    );
    return () => window.clearTimeout(timeout);
  }, [key, onDone]);

  return (
    <div
      aria-hidden="true"
      className="absolute"
      style={{
        left: box.x - box.width / 2,
        top: box.y - box.height / 2,
        width: box.width,
        height: box.height,
        rotate: `${box.rotation}deg`,
      }}
    >
      {shards.map((shard, index) => (
        <motion.div
          key={index}
          className="absolute inset-0"
          style={{
            clipPath: shard.clipPath,
            transformOrigin: `${shard.centre.x * 100}% ${shard.centre.y * 100}%`,
            backgroundImage: `${SHIELD_TINT.background}, url(${cardImageUrl(imageId)})`,
            backgroundSize: "cover",
            borderRadius: SHIELD_TINT.borderRadius,
          }}
          initial={{ x: 0, y: 0, rotate: 0, opacity: 1 }}
          animate={{
            x: shard.dx * box.width,
            y: shard.dy * box.height,
            rotate: shard.rotate,
            opacity: 0,
          }}
          transition={{
            duration: SHATTER_SECONDS,
            delay: shard.delay,
            ease: [0.2, 0.8, 0.4, 1],
            opacity: {
              duration: SHATTER_SECONDS,
              delay: shard.delay,
              ease: "easeIn",
            },
          }}
          onAnimationComplete={
            index === shards.length - 1 ? () => onDone(key) : undefined
          }
        />
      ))}
      <motion.div
        className="pointer-events-none absolute aspect-square w-full rounded-full"
        style={{
          left: `${(impact.x - 0.5) * 100}%`,
          top: `calc(${impact.y * 100}% - ${box.width / 2}px)`,
          background:
            "radial-gradient(circle, rgba(255, 255, 255, 0.95) 0%, rgba(191, 219, 254, 0.6) 35%, rgba(59, 130, 246, 0) 70%)",
        }}
        initial={{ scale: 0.3, opacity: 1 }}
        animate={{ scale: 1.6, opacity: 0 }}
        transition={{ duration: SHATTER_SECONDS / 2, ease: "easeOut" }}
      />
    </div>
  );
}

/** A copy of a card that acted, on top of it, growing as it fades. */
function CardEcho({
  echo,
  onDone,
}: {
  echo: ActiveEcho;
  onDone: (key: number) => void;
}) {
  const { key, imageId, box, delay } = echo;

  // Removed on a timer as well, so the copy can never linger on screen.
  useEffect(() => {
    const timeout = window.setTimeout(
      () => onDone(key),
      (delay + ECHO_SECONDS) * 1000 + LANDING_GRACE_MS,
    );
    return () => window.clearTimeout(timeout);
  }, [key, delay, onDone]);

  return (
    <motion.img
      src={cardImageUrl(imageId)}
      alt=""
      draggable={false}
      className="absolute"
      style={{
        left: box.x - box.width / 2,
        top: box.y - box.height / 2,
        width: box.width,
        height: box.height,
        borderRadius: "5%",
        rotate: box.rotation,
        ...CARD_IMAGE_PLACEHOLDER,
      }}
      // Unseen while it waits for the card to land.
      initial={{ opacity: 0, scale: 1 }}
      animate={{ opacity: [ECHO_OPACITY, 0], scale: [1, ECHO_SCALE] }}
      transition={{ duration: ECHO_SECONDS, delay, ease: "easeOut" }}
      onAnimationComplete={() => onDone(key)}
    />
  );
}
