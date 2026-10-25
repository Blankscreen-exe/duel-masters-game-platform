import { describe, expect, it } from "vitest";
import {
  callMessage,
  isCoinTossMessage,
  nextCoinToss,
  turnMessage,
  type CoinTossState,
} from "./coinTossState";

const alice = { uid: "a", username: "Alice" };
const bob = { uid: "b", username: "Bob" };

describe("the coin toss", () => {
  it("knows its own messages", () => {
    expect(isCoinTossMessage({ header: "choose_toss" })).toBe(true);
    expect(isCoinTossMessage({ header: "toss_result" })).toBe(true);
    expect(isCoinTossMessage({ header: "state_update" })).toBe(false);
    expect(isCoinTossMessage({})).toBe(false);
  });

  it("asks this player to call, or says who calls", () => {
    expect(nextCoinToss(null, { header: "choose_toss", player: alice })).toEqual({
      step: "calling",
      caller: alice,
      mine: true,
    });
    expect(
      nextCoinToss(null, { header: "toss_being_chosen", player: alice }),
    ).toEqual({ step: "calling", caller: alice, mine: false });
  });

  it("shows the result, then who chooses to play first", () => {
    const tossed = nextCoinToss(
      { step: "calling", caller: alice, mine: false },
      { header: "toss_result", caller: alice, call: "heads", outcome: "tails", winner: bob },
    );
    expect(tossed).toEqual({
      step: "tossed",
      caller: alice,
      call: "heads",
      outcome: "tails",
      winner: bob,
      mine: null,
    });

    expect(nextCoinToss(tossed, { header: "choose_turn", player: bob })).toEqual({
      ...(tossed as CoinTossState),
      mine: true,
    });
    expect(
      nextCoinToss(tossed, { header: "turn_being_chosen", player: bob }),
    ).toEqual({ ...(tossed as CoinTossState), mine: false });
  });

  it("ignores a choice arriving before the result", () => {
    const calling: CoinTossState = { step: "calling", caller: alice, mine: true };
    expect(nextCoinToss(calling, { header: "choose_turn", player: alice })).toBe(calling);
    expect(nextCoinToss(null, { header: "choose_turn", player: alice })).toBeNull();
  });

  it("speaks the engine's messages", () => {
    expect(callMessage("heads")).toEqual({ header: "toss_chosen", prediction: 1 });
    expect(callMessage("tails")).toEqual({ header: "toss_chosen", prediction: -1 });
    expect(turnMessage(true)).toEqual({ header: "turn_chosen", player: 1 });
    expect(turnMessage(false)).toEqual({ header: "turn_chosen", player: -1 });
  });
});
