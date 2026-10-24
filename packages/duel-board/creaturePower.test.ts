import { describe, expect, it } from "vitest";
import {
  attackBonus,
  describePower,
  findCreaturePower,
  formatPowerDelta,
  lowerEdgeShare,
  powerChanges,
  powerPopupPoint,
  powerReadout,
  powerTone,
} from "./creaturePower";
import { CARD_ASPECT_RATIO } from "./cardImage";
import type { CreaturePower } from "./types";
import { card, matchState, measuredCard } from "./test/fixtures";

const power = (
  printed: number,
  current = printed,
  attacking = current,
): CreaturePower => ({
  printed,
  current,
  attacking,
});

describe("powerTone", () => {
  it("tells raised, lowered and printed power apart", () => {
    expect(powerTone(power(2000, 5000))).toBe("raised");
    expect(powerTone(power(5000, 2000))).toBe("lowered");
    expect(powerTone(power(3000))).toBe("printed");
  });
});

describe("attackBonus", () => {
  it("is the extra power while attacking", () => {
    expect(attackBonus(power(2000, 2000, 4000))).toBe(2000);
  });

  it("is never negative", () => {
    expect(attackBonus(power(2000, 3000, 2000))).toBe(0);
  });
});

describe("powerReadout", () => {
  it("shows the printed power when nothing has changed it", () => {
    expect(powerReadout(power(3000), false)).toEqual({
      shown: 3000,
      tone: "printed",
    });
  });

  it("shows changed power in its tone", () => {
    expect(powerReadout(power(3000, 4000), false)).toMatchObject({
      shown: 4000,
      tone: "raised",
    });
    expect(powerReadout(power(3000, 1000), false)).toMatchObject({
      shown: 1000,
      tone: "lowered",
    });
  });

  it("leaves out the bonus while the creature is not attacking", () => {
    expect(powerReadout(power(3000, 4000, 6000), false)).toEqual({
      shown: 4000,
      tone: "raised",
    });
  });

  it("shows the attacking power while the creature attacks", () => {
    expect(powerReadout(power(3000, 3000, 5000), true)).toEqual({
      shown: 5000,
      tone: "attacking",
    });
  });
});

describe("formatPowerDelta", () => {
  it("signs the change", () => {
    expect(formatPowerDelta(2000)).toBe("+2000");
    expect(formatPowerDelta(-3000)).toBe("−3000");
  });
});

describe("describePower", () => {
  it("names only the current power when nothing changed", () => {
    expect(describePower(power(3000))).toBe("Power 3000");
  });

  it("names the printed power and the attacking power when they differ", () => {
    expect(describePower(power(2000, 5000, 7000))).toBe(
      "Power 5000, printed 2000, 7000 when attacking",
    );
  });
});

describe("findCreaturePower", () => {
  it("finds a creature's power on either side", () => {
    const state = matchState(
      { powers: { mine: power(1000) } },
      { powers: { theirs: power(2000) } },
    );

    expect(findCreaturePower(state, "mine")).toEqual(power(1000));
    expect(findCreaturePower(state, "theirs")).toEqual(power(2000));
    expect(findCreaturePower(state, "gone")).toBeNull();
  });
});

describe("powerChanges", () => {
  it("reports a creature whose power changed", () => {
    const previous = matchState({
      playzone: [card("a")],
      powers: { a: power(2000) },
    });
    const next = matchState({
      playzone: [card("a")],
      powers: { a: power(2000, 5000) },
    });

    expect(powerChanges(previous, next)).toEqual([
      { virtualId: "a", side: "me", delta: 3000 },
    ]);
  });

  it("reports lowered power on the opponent's side", () => {
    const previous = matchState({}, { powers: { a: power(5000) } });
    const next = matchState({}, { powers: { a: power(5000, 2000) } });

    expect(powerChanges(previous, next)).toEqual([
      { virtualId: "a", side: "opponent", delta: -3000 },
    ]);
  });

  it("ignores a creature that just arrived", () => {
    const previous = matchState();
    const next = matchState({ powers: { a: power(2000, 5000) } });

    expect(powerChanges(previous, next)).toEqual([]);
  });

  it("ignores attacking power coming and going", () => {
    const previous = matchState({ powers: { a: power(2000) } });
    const next = matchState({ powers: { a: power(2000, 2000, 4000) } });

    expect(powerChanges(previous, next)).toEqual([]);
  });
});

describe("powerPopupPoint", () => {
  it("is just above an upright card", () => {
    const [, upright] = measuredCard("a", 100, 300);

    expect(powerPopupPoint(upright)).toEqual({ x: 100, y: 300 - 84 / 2 });
  });

  it("is just above a card lying on its side", () => {
    const [, tapped] = measuredCard("a", 100, 300, { rotation: -90 });

    expect(powerPopupPoint(tapped)).toEqual({ x: 100, y: 300 - 60 / 2 });
  });
});

describe("lowerEdgeShare", () => {
  it("is the bottom of the box for an upright card", () => {
    expect(lowerEdgeShare(false)).toBe(0);
  });

  it("leaves the gap below a card lying on its side", () => {
    expect(lowerEdgeShare(true)).toBeCloseTo((1 - CARD_ASPECT_RATIO) / 2);
  });
});
