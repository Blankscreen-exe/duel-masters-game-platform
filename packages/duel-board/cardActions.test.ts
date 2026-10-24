import { describe, expect, it } from "vitest";
import { cardActions, findBoardCard, isFaceUp } from "./cardActions";
import { PLAYABLE_FLAG, SUMMONING_SICK_FLAG, TAP_ABILITY_FLAG } from "./types";
import { card, matchState, shield } from "./test/fixtures";

const ids = (actions: { id: string }[]) => actions.map((action) => action.id);

describe("findBoardCard", () => {
  const state = matchState(
    {
      hand: [card("h")],
      playzone: [card("p")],
      manazone: [card("m")],
      shieldzone: [shield("s")],
    },
    {
      playzone: [card("op")],
      manazone: [card("om")],
      shieldzone: [shield("os")],
    },
  );

  it.each([
    ["h", "hand"],
    ["p", "myPlayzone"],
    ["m", "myManazone"],
    ["s", "myShieldzone"],
    ["op", "opponentPlayzone"],
    ["om", "opponentManazone"],
    ["os", "opponentShieldzone"],
  ])("finds %s in %s", (virtualId, zone) => {
    expect(findBoardCard(state, virtualId)?.zone).toBe(zone);
  });

  it("does not find a card that left the board", () => {
    expect(findBoardCard(state, "gone")).toBeNull();
  });
});

describe("cardActions", () => {
  it("offers summoning, mana and viewing for a creature in hand", () => {
    const creature = card("a", PLAYABLE_FLAG);
    const actions = cardActions({ card: creature, zone: "hand" }, matchState());

    expect(actions).toEqual([
      { id: "play", label: "Summon", disabledReason: null },
      { id: "charge", label: "Add to mana", disabledReason: null },
      { id: "view", label: "View", disabledReason: null },
    ]);
  });

  it("names playing a spell casting", () => {
    const spell = card("a", PLAYABLE_FLAG, "spell");
    const [play] = cardActions({ card: spell, zone: "hand" }, matchState());

    expect(play.label).toBe("Cast");
  });

  it("falls back to a neutral label when the type is unknown", () => {
    const unknown = card("a", PLAYABLE_FLAG, "");
    const [play] = cardActions({ card: unknown, zone: "hand" }, matchState());

    expect(play.label).toBe("Play");
  });

  it("explains a card that cannot be afforded", () => {
    const [play] = cardActions({ card: card("a"), zone: "hand" }, matchState());

    expect(play.disabledReason).toBe(
      "You don't have enough mana to play this card",
    );
  });

  it("explains that the attack step closed summoning and casting", () => {
    const attacking = matchState(
      {},
      {},
      { step: "attack", hasAttackedThisRound: true },
    );

    const [summon, charge] = cardActions(
      { card: card("a", PLAYABLE_FLAG), zone: "hand" },
      attacking,
    );
    const [cast] = cardActions(
      { card: card("b", PLAYABLE_FLAG, "spell"), zone: "hand" },
      attacking,
    );

    expect(summon.disabledReason).toBe(
      "You can't summon creatures after attacking or using a tap ability",
    );
    expect(cast.disabledReason).toBe(
      "You can't cast spells after attacking or using a tap ability",
    );
    expect(charge.disabledReason).toBe(
      "You can't charge mana after attacking or using a tap ability",
    );
  });

  it("explains mana already added", () => {
    const [, charge] = cardActions(
      { card: card("a"), zone: "hand" },
      matchState({}, {}, { hasAddedManaThisRound: true }),
    );

    expect(charge.disabledReason).toBe(
      "You've already added mana to your manazone this turn",
    );
  });

  it("explains mana closed by playing a card", () => {
    const [, charge] = cardActions(
      { card: card("a"), zone: "hand" },
      matchState({}, {}, { step: "main", canChargeManaThisRound: false }),
    );

    expect(charge.disabledReason).toBe(
      "You can't charge mana after playing creatures or spells",
    );
  });

  it("offers attacks for a creature in the battle zone", () => {
    const actions = cardActions(
      { card: card("a"), zone: "myPlayzone" },
      matchState(),
    );

    expect(ids(actions)).toEqual(["attackPlayer", "attackCreature", "view"]);
  });

  it("adds the tap ability when the creature has one", () => {
    const actions = cardActions(
      { card: card("a", TAP_ABILITY_FLAG), zone: "myPlayzone" },
      matchState(),
    );

    expect(ids(actions)).toEqual([
      "attackPlayer",
      "attackCreature",
      "tapAbility",
      "view",
    ]);
  });

  it("explains that a creature summoned this turn cannot attack yet", () => {
    const actions = cardActions(
      {
        card: card("a", SUMMONING_SICK_FLAG | TAP_ABILITY_FLAG),
        zone: "myPlayzone",
      },
      matchState(),
    );

    expect(
      actions.map(({ id, disabledReason }) => [id, disabledReason]),
    ).toEqual([
      [
        "attackPlayer",
        "Summoning sickness: it can't attack until your next turn",
      ],
      [
        "attackCreature",
        "Summoning sickness: it can't attack until your next turn",
      ],
      [
        "tapAbility",
        "Summoning sickness: it can't use its tap ability until your next turn",
      ],
      ["view", null],
    ]);
  });

  it("only offers viewing on the opponent's turn", () => {
    const theirTurn = matchState({}, {}, { myTurn: false });

    expect(
      ids(cardActions({ card: card("a"), zone: "hand" }, theirTurn)),
    ).toEqual(["view"]);
    expect(
      ids(cardActions({ card: card("b"), zone: "myPlayzone" }, theirTurn)),
    ).toEqual(["view"]);
  });

  it("only offers viewing to a spectator", () => {
    const watching = matchState({}, {}, { spectator: true });

    expect(
      ids(cardActions({ card: card("a"), zone: "myPlayzone" }, watching)),
    ).toEqual(["view"]);
  });

  it("only offers viewing for cards in other zones", () => {
    for (const zone of [
      "myManazone",
      "opponentPlayzone",
      "opponentManazone",
    ] as const) {
      expect(ids(cardActions({ card: card("a"), zone }, matchState()))).toEqual(
        ["view"],
      );
    }
  });

  it("offers nothing for a face down card", () => {
    expect(
      cardActions(
        { card: shield("s"), zone: "opponentShieldzone" },
        matchState(),
      ),
    ).toEqual([]);
  });
});

describe("isFaceUp", () => {
  it("is false for a face down shield and the card back", () => {
    expect(isFaceUp(shield("s"))).toBe(false);
    expect(isFaceUp({ ...card("a"), uid: "backside" })).toBe(false);
  });

  it("is true for a card with its image", () => {
    expect(isFaceUp(card("a"))).toBe(true);
  });
});
