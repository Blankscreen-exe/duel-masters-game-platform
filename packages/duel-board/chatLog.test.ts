import { describe, expect, it } from "vitest";
import { cardSide, logKind, logSegments } from "./chatLog";
import type { ChatCard } from "./types";

describe("logKind", () => {
  // Lines as the server writes them.
  it.each([
    ["Your turn, Player1", "turnStart"],
    ["Player1 ended their turn", "turnEnd"],
    ["Player2 summoned Aqua Vehicle to the battle zone", "summon"],
    ["Player1 casted the spell Terror Pit", "spell"],
    ["2 of Player1's shields were broken", "shieldBreak"],
    ["Player2 drew 1 card", "draw"],
    [
      "Player2 drew a card when Vorg attacked due to Gale's survivor ability",
      "draw",
    ],
    ["Player1 shuffled their deck", "shuffle"],
    ["Vorg was discarded from Player1's hand by Gigargon", "discard"],
    ["Vorg (2000) was destroyed by Aqua Vehicle (1000)", "destroy"],
    [
      "Vorg was destroyed by Natural Snare and moved to the mana zone",
      "destroy",
    ],
    ["Vorg was returned to Player2's hand by Aqua Surfer", "toHand"],
    [
      "Vorg was returned to hand instead of being destroyed due to X's effect.",
      "toHand",
    ],
    ["Vorg was moved to manazone instead of being destroyed.", "toMana"],
    ["Ruby Grass was added to Player2's manazone", "toMana"],
    ["Ruby Grass was put in the mana zone instead of your graveyard", "toMana"],
    ["Ruby Grass was put from Player1's hand to his mana zone.", "toMana"],
    ["Solar Ray effect: Ruby Grass moved from MZ to GY", "toGraveyard"],
    ["A shield was put into Player2's graveyard by Bone Spider", "toGraveyard"],
    [
      "Vorg was moved to Player1's graveyard instead of hand by X",
      "toGraveyard",
    ],
    [
      "Player1 put the top card of his deck into the shieldzone from X's effect",
      "toShields",
    ],
    ["Vorg was added to Player1's shields by Rainbow Stone", "toShields"],
    [
      "Vorg was shuffled into Player2's deck instead of being destroyed",
      "shuffle",
    ],
    ["Player1 retrieved Vorg from their graveyard", "toHand"],
    ["Vorg was untapped by Gale's survivor ability", "untap"],
    ["Vorg was tapped by Holy Awe", "tap"],
    ["Player1 activated Vorg's tap ability", "tap"],
    ["Vorg was given +3000 power until the end of the turn", "boost"],
    ["Vorg was turned face up from Player1's shieldzone.", "effect"],
  ])("%s", (line, kind) => {
    expect(logKind(line)).toBe(kind);
  });
});

describe("logSegments", () => {
  const vorg: ChatCard = { name: "Vorg", uid: "vorg", owner: "Player1" };
  const vehicle: ChatCard = { name: "Aqua Vehicle", uid: "aqua" };

  it("splits a line around the cards it names", () => {
    expect(
      logSegments("Vorg (2000) was destroyed by Aqua Vehicle (1000)", [
        vorg,
        vehicle,
      ]),
    ).toEqual([
      { text: "Vorg", card: vorg },
      { text: " (2000) was destroyed by " },
      { text: "Aqua Vehicle", card: vehicle },
      { text: " (1000)" },
    ]);
  });

  it("tells two mentions of one name apart by their order", () => {
    const theirs: ChatCard = { name: "Vorg", uid: "vorg", owner: "Player2" };

    expect(
      logSegments("Vorg was destroyed by Vorg", [vorg, theirs])
        .filter((segment) => segment.card)
        .map((segment) => segment.card?.owner),
    ).toEqual(["Player1", "Player2"]);
  });

  it("leaves a line without cards as it is", () => {
    expect(logSegments("Your turn, Player1")).toEqual([
      { text: "Your turn, Player1" },
    ]);
  });

  it("keeps the text when a name is not where the server said", () => {
    expect(logSegments("Player1 drew 1 card", [vorg])).toEqual([
      { text: "Player1 drew 1 card" },
    ]);
  });
});

describe("cardSide", () => {
  const players = { me: "Player1", opponent: "Player2" };

  it("tells my cards from my opponent's", () => {
    expect(cardSide({ name: "A", uid: "a", owner: "Player1" }, players)).toBe(
      "mine",
    );
    expect(cardSide({ name: "A", uid: "a", owner: "Player2" }, players)).toBe(
      "theirs",
    );
  });

  it("does not guess when the owner is unknown", () => {
    expect(cardSide({ name: "A", uid: "a" }, players)).toBe("unknown");
    expect(cardSide({ name: "A", uid: "a", owner: "Player1" }, undefined)).toBe(
      "unknown",
    );
  });
});
