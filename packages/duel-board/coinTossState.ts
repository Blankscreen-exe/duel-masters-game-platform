/** A player taking part in the coin toss, as the server names them. */
export interface TossPlayer {
  uid: string;
  username: string;
}

export type CoinSide = "heads" | "tails";

/** Where the coin toss that decides who plays first has got to, as this
 * screen sees it. */
export type CoinTossState =
  /** Someone is calling heads or tails: this player, or someone else. */
  | { step: "calling"; caller: TossPlayer; mine: boolean }
  /** The coin has been tossed: it shows the result, then the winner chooses
   * whether to play first, this player or someone else. */
  | {
      step: "tossed";
      caller: TossPlayer;
      call: CoinSide;
      outcome: CoinSide;
      winner: TossPlayer;
      /** Whether this player chooses; null until the server says who does. */
      mine: boolean | null;
    };

/** A coin toss message from the server. */
export type CoinTossMessage =
  | { header: "choose_toss" | "toss_being_chosen"; player: TossPlayer }
  | {
      header: "toss_result";
      caller: TossPlayer;
      call: CoinSide;
      outcome: CoinSide;
      winner: TossPlayer;
    }
  | { header: "choose_turn" | "turn_being_chosen"; player: TossPlayer };

const TOSS_HEADERS = new Set([
  "choose_toss",
  "toss_being_chosen",
  "toss_result",
  "choose_turn",
  "turn_being_chosen",
]);

/** Whether a message from the server is part of the coin toss. */
export function isCoinTossMessage(message: {
  header?: unknown;
}): message is CoinTossMessage {
  return typeof message.header === "string" && TOSS_HEADERS.has(message.header);
}

/** The coin toss after a message from the server. */
export function nextCoinToss(
  current: CoinTossState | null,
  message: CoinTossMessage,
): CoinTossState | null {
  switch (message.header) {
    case "choose_toss":
    case "toss_being_chosen":
      return {
        step: "calling",
        caller: message.player,
        mine: message.header === "choose_toss",
      };
    case "toss_result":
      return {
        step: "tossed",
        caller: message.caller,
        call: message.call,
        outcome: message.outcome,
        winner: message.winner,
        mine: null,
      };
    case "choose_turn":
    case "turn_being_chosen":
      // Only after the result, which always comes first.
      if (current?.step !== "tossed") {
        return current;
      }
      return { ...current, mine: message.header === "choose_turn" };
  }
}

/** The message calling the toss. */
export function callMessage(side: CoinSide) {
  return { header: "toss_chosen", prediction: side === "heads" ? 1 : -1 };
}

/** The message choosing to play first or second. */
export function turnMessage(playFirst: boolean) {
  return { header: "turn_chosen", player: playFirst ? 1 : -1 };
}
