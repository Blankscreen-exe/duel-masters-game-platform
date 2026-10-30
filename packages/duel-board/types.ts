import type { ReactNode } from "react";

// Message is the default message struct
export interface Message {
  header: string;
}

// DecksMessage lists the users decks
export interface DecksMessage {
  header: string;
  decks: LegacyDeck[];
}

export interface LegacyDeck {
  uid: string;
  owner: string;
  name: string;
  public: boolean;
  standard: boolean;
  cards: string[];
}

// ChatMessage stores information about a chat message
export interface ChatMessage {
  header: string;
  message: string;
  sender: string;
  color: string;
  /** The cards a game log line names, in the order it names them. */
  cards?: ChatCard[];
}

/** A card a game log line names. */
export interface ChatCard {
  /** The card's name as the line writes it. */
  name: string;
  /** The card's image id, which shows it. */
  uid: string;
  /** The username of the player whose card it is, when it can be told. */
  owner?: string;
  /** The card's id while it is where both players can see it. */
  virtualId?: string;
}

export interface DuelChatUser {
  username: string;
  avatarUrl?: string | null;
}

/**
 * Starts a non-critical lookup for an unknown sender. Duel calls it at most
 * once per normalized username and renders without waiting for it.
 */
export type DuelChatUserResolver = (
  username: string,
) => Promise<DuelChatUser | null | undefined>;

export interface DuelChatUserTriggerProps {
  user: DuelChatUser;
  children: ReactNode;
  className: string;
}

/** Synchronously wraps an avatar or username; do not perform lookups here. */
export type DuelChatUserTriggerRenderer = (
  props: DuelChatUserTriggerProps,
) => ReactNode;

export const TAPPED_FLAG = 1;
export const PLAYABLE_FLAG = 2;
export const TAP_ABILITY_FLAG = 4;
export const SHIELD_FACE_UP_FLAG = 8;
/** A creature that entered the battle zone this turn and cannot attack or use
 * a tap ability yet. */
export const SUMMONING_SICK_FLAG = 16;

export const cardHasFlag = (flags: number, flag: number): boolean => {
  return (flags & flag) > 0;
};

export type CardType = "creature" | "spell";

// CardState stores information about the state of a card
export interface CardState {
  virtualId: string;
  uid: string;
  name: string;
  civilization: string;
  flags: number;
  /** Empty when the card is face down. */
  type: CardType | "";
}

// ShieldState stores information about the state of a shield
export interface ShieldState {
  virtualId: string;
  uid: string;
  flags: number;
}

// PlayerState stores information about the state of the current player
export interface PlayerState {
  username: string;
  color: string;
  deck: number;
  handCount: number;
  hand: CardState[];
  shieldzone: ShieldState[];
  shieldMap: Record<string, number>;
  manazone: CardState[];
  graveyard: CardState[];
  playzone: CardState[];
  /** The power of each creature in the battle zone, by virtualId. */
  powers: Record<string, CreaturePower>;
  /** The abilities and limits of each creature in the battle zone, by
   * virtualId. */
  statuses: Record<string, CreatureStatus[]>;
}

/** An ability or limit a creature in the battle zone has right now. */
export interface CreatureStatus {
  /** Named after the condition behind it, such as "blocker". */
  id: string;
  /** The amount it carries, such as a power attacker's bonus. */
  value?: number;
  /** Given by another card's effect rather than printed on the card. */
  granted?: boolean;
}

/** A creature's power as the rules see it right now. Effects can raise or
 * lower it from the power printed on the card. */
export interface CreaturePower {
  printed: number;
  current: number;
  /** The power while the creature attacks, which power attacker raises. */
  attacking: number;
}

/** The steps of a turn in order, as named by the server. A turn only waits for
 * its player in charge, main and attack, and goes straight from charge to attack
 * when the player plays nothing. */
export type TurnStep =
  | "begin_turn"
  | "untap"
  | "start_of_turn"
  | "draw"
  | "charge"
  | "main"
  | "attack"
  | "end"
  | "end_of_turn";

// MatchState stores information about the current state of the match in the eyes of a given player
export interface MatchState {
  myTurn: boolean;
  hasAddedManaThisRound: boolean;
  hasAttackedThisRound: boolean;
  canChargeManaThisRound: boolean;
  me: PlayerState;
  opponent: PlayerState;
  spectator: boolean;
  /** Empty before the first turn has begun. */
  step: TurnStep | "";
  /** The shields broken since the last state, by virtualId. */
  brokenShields: string[];
  /** The cards that acted since the last state, by virtualId: summoned, cast,
   * attacked, blocked, used a tap ability, or moved or destroyed another card
   * with their effect. */
  activatedCards: string[];
  /** Whether the player whose turn it is has put a card into their mana zone
   * this turn, which they may only do once. */
  turnManaCharged: boolean;
  /** The creature whose attack is under way, by virtualId, from when the
   * attack goes ahead until it has resolved. */
  attacker?: string;
  /** The number of the turn under way, counting both players' turns and extra
   * turns from 1. 0 before the first turn has begun. */
  turn: number;
}

// MatchStateMessage is the message that should be sent to the client for state updates
export interface MatchStateMessage {
  header: string;
  state: MatchState;
}

// WarningMessage is used to send a warning to a player
export interface WarningMessage {
  header: string;
  message: string;
}

export interface DuelFinishedPlayer {
  uid: string;
  username?: string;
}

/** How a game ended: won by the game's rules, by surrender, by a player not
 * coming back, or stopped without a winner, such as by the server. */
export type DuelEnding = "play" | "surrender" | "disconnect" | "aborted";

export interface DuelFinishedMessage {
  header: "duel_finished";
  duelId: string;
  /** Absent when there is none. */
  winner?: DuelFinishedPlayer;
  ending: DuelEnding;
  /** False for a game too short to count, stopped, or practice. */
  counted: boolean;
  /** Set for a game that never counts, such as one against the computer. */
  practice: boolean;
  turns: number;
  durationSeconds: number;
}

export enum ActionType {
  None = "",
  Count = "count",
  Question = "question",
  Order = "order",
  Searchable = "searchable",
  ShowCards = "show_cards",
}

// ActionMessage is used to prompt the user to make a selection of the specified cards
/** The kind of a question, as the engine tells it (match.PromptKind). */
export type PromptKind =
  | "effect"
  | "pay_mana"
  | "attack_target"
  | "confirm_attack"
  | "break_shields"
  | "block"
  | "shield_trigger"
  | "evolve"
  | "tap_ability";

/** The card whose effect asks a question. */
export interface PromptSource {
  virtualId: string;
  uid: string;
  name: string;
}

/** What a question is about, for programs that answer as a player does. */
export interface PromptAbout {
  kind?: PromptKind;
  /** Told only when the player asked may know the card. */
  source?: PromptSource;
  /** The offered cards, by virtualId, that are the opponent's. */
  opponentCards?: string[];
}

export interface ActionMessage extends PromptAbout {
  header: string;
  /** The question's number, which its answer names. Absent from a preview
   * that only shows cards and waits for nothing. */
  prompt?: number;
  actionType: ActionType;
  cards?: CardState[] | Record<string, CardState[]>;
  text: string;
  minSelections?: number;
  maxSelections?: number;
  cancellable?: boolean;
  unselectableCards?: CardState[];
  choices?: string[];
  showCards?: {
    cards: string[];
    dismissable: boolean;
  };
}

// MultipartActionMessage is used to prompt the user to make a selection of the specified cards
export interface MultipartActionMessage extends PromptAbout {
  header: string;
  cards: Record<string, CardState[]>;
  text: string;
  minSelections: number;
  maxSelections: number;
  cancellable: boolean;
}

// ActionWarningMessage is used to apply an error
export interface ActionWarningMessage {
  header: string;
  message: string;
}

// WaitMessage is used to send a waiting popup with a message
export interface WaitMessage {
  header: string;
  message: string;
}

// LobbyChatMessage is used to store chat messages
export interface LobbyChatMessage {
  username: string;
  color: string;
  message: string;
  timestamp: number;
  removed?: boolean;
}

// LobbyChatMessages is used to store chat messages
export interface LobbyChatMessages {
  header: string;
  messages: LobbyChatMessage[];
}

// UserMessage holds information about users
export interface UserMessage {
  username: string;
  color: string;
  hub: string;
  permissions: string[];
}

// UserListMessage is used to send a list of online users
export interface UserListMessage {
  header: string;
  users: UserMessage[];
}

// MatchMessage holds information about a match
export interface MatchMessage {
  id: string;
  p1: string;
  p1color: string;
  p2: string;
  p2color: string;
  name: string;
  spectate: boolean;
  matchmaking: boolean;
  format: string;
}

// MatchesListMessage is used to list open matches
export interface MatchesListMessage {
  header: string;
  matches: MatchMessage[];
}

export interface MatchRequestMessage {
  id: string;
  name: string;
  host_id: string;
  host_name: string;
  host_color: string;
  guest_id: string;
  guest_name: string;
  guest_color: string;
  format: string;
  link_code: string;
}

export interface MatchRequestsListMessage {
  header: string;
  requests: MatchRequestMessage[];
}

export interface MatchForwardMessage {
  header: string;
  id: string;
}

// ShowCardsMessage is used to show the user n cards without an action to perform
export interface ShowCardsMessage {
  header: string;
  /** Set when the preview is a question that must be closed. */
  prompt?: number;
  message: string;
  cards: string[];
}

export interface PinnedMessages {
  header: string;
  messages: string[];
}

export interface PlaySoundMessage {
  header: string;
  sound: string;
}
