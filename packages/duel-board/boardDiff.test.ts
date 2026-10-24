import { describe, expect, it } from "vitest";
import { diffBoard, locateCards } from "./boardDiff";
import { TAPPED_FLAG } from "./types";
import { card, matchState, shield } from "./test/fixtures";

describe("locateCards", () => {
  it("finds every listed card on both sides", () => {
    const state = matchState(
      {
        hand: [card("h")],
        playzone: [card("p")],
        manazone: [card("m")],
        shieldzone: [shield("s")],
        graveyard: [card("g")],
      },
      { playzone: [card("op")], shieldzone: [shield("os")] },
    );

    expect(Object.fromEntries(locateCards(state))).toEqual({
      h: { side: "me", zone: "hand" },
      p: { side: "me", zone: "playzone" },
      m: { side: "me", zone: "manazone" },
      s: { side: "me", zone: "shieldzone" },
      g: { side: "me", zone: "graveyard" },
      op: { side: "opponent", zone: "playzone" },
      os: { side: "opponent", zone: "shieldzone" },
    });
  });

  it("has nothing for an empty board", () => {
    expect(locateCards(matchState()).size).toBe(0);
  });
});

describe("diffBoard", () => {
  it("reports nothing when no card changed zone", () => {
    const previous = matchState({ hand: [card("a")], playzone: [card("b")] });
    const next = matchState({ hand: [card("a")], playzone: [card("b")] });

    expect(diffBoard(previous, next)).toEqual([]);
  });

  it("ignores a card tapped in place", () => {
    const previous = matchState({ playzone: [card("a")] });
    const next = matchState({ playzone: [card("a", TAPPED_FLAG)] });

    expect(diffBoard(previous, next)).toEqual([]);
  });

  it("ignores cards reordered within a zone", () => {
    const previous = matchState({ manazone: [card("a"), card("b")] });
    const next = matchState({ manazone: [card("b"), card("a")] });

    expect(diffBoard(previous, next)).toEqual([]);
  });

  it("reports a summon from hand to the battle zone", () => {
    const previous = matchState({ hand: [card("a"), card("b")] });
    const next = matchState({ hand: [card("b")], playzone: [card("a")] });

    expect(diffBoard(previous, next)).toEqual([
      {
        virtualId: "a",
        from: { side: "me", zone: "hand" },
        to: { side: "me", zone: "playzone" },
      },
    ]);
  });

  it("reports a broken shield arriving in its owner's hand", () => {
    const previous = matchState({ shieldzone: [shield("s")] });
    const next = matchState({ hand: [card("s")] });

    expect(diffBoard(previous, next)).toEqual([
      {
        virtualId: "s",
        from: { side: "me", zone: "shieldzone" },
        to: { side: "me", zone: "hand" },
      },
    ]);
  });

  it("reports a drawn card as coming from out of sight", () => {
    const previous = matchState();
    const next = matchState({ hand: [card("a")] });

    expect(diffBoard(previous, next)).toEqual([
      { virtualId: "a", from: null, to: { side: "me", zone: "hand" } },
    ]);
  });

  it("reports a card the opponent plays from their hidden hand", () => {
    const previous = matchState();
    const next = matchState({}, { playzone: [card("a")] });

    expect(diffBoard(previous, next)).toEqual([
      {
        virtualId: "a",
        from: null,
        to: { side: "opponent", zone: "playzone" },
      },
    ]);
  });

  it("reports a card going out of sight", () => {
    const previous = matchState({}, { shieldzone: [shield("s")] });
    const next = matchState();

    expect(diffBoard(previous, next)).toEqual([
      {
        virtualId: "s",
        from: { side: "opponent", zone: "shieldzone" },
        to: null,
      },
    ]);
  });

  it("reports several moves from one update", () => {
    const previous = matchState(
      { playzone: [card("attacker")] },
      { playzone: [card("blocker")], shieldzone: [shield("s")] },
    );
    const next = matchState(
      { graveyard: [card("attacker")] },
      { playzone: [card("blocker")] },
    );

    expect(diffBoard(previous, next)).toEqual(
      expect.arrayContaining([
        {
          virtualId: "attacker",
          from: { side: "me", zone: "playzone" },
          to: { side: "me", zone: "graveyard" },
        },
        {
          virtualId: "s",
          from: { side: "opponent", zone: "shieldzone" },
          to: null,
        },
      ]),
    );
    expect(diffBoard(previous, next)).toHaveLength(2);
  });
});
