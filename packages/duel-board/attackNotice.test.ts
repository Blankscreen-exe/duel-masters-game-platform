import { describe, expect, it } from "vitest";
import { blockedAttack, describeAttack, plannedAttack } from "./attackNotice";
import { card, matchState, shield } from "./test/fixtures";

describe("plannedAttack", () => {
  const state = (shields = [shield("s")]) => {
    return matchState(
      {
        playzone: [card("a")],
        powers: { a: { printed: 2000, current: 3000, attacking: 5000 } },
      },
      { shieldzone: shields },
    );
  };

  it("shows the power the creature attacks with, not its power now", () => {
    expect(plannedAttack(state(), "a", "player")).toEqual({
      side: "mine",
      attacker: { name: "Card a", power: 5000 },
      target: { name: "Opponent's shields" },
    });
  });

  it("aims at the opponent once their shields are gone", () => {
    expect(plannedAttack(state([]), "a", "player")?.target.name).toBe(
      "Opponent",
    );
  });

  it("aims at a creature still to be chosen", () => {
    expect(plannedAttack(state(), "a", "creature")?.target).toEqual({
      name: "Opponent's creature",
    });
  });

  it("is null for a creature that has left the battle zone", () => {
    expect(plannedAttack(state(), "gone", "player")).toBeNull();
  });
});

describe("blockedAttack", () => {
  const attacker = { name: "Bolshack Dragon", power: 7000 };

  it("aims at the viewer when no creature is attacked", () => {
    expect(blockedAttack({ attacker })).toEqual({
      side: "theirs",
      attacker,
      target: { name: "You" },
    });
  });

  it("names the creature attacked and its power", () => {
    const target = { name: "Aqua Hulcus", power: 2000 };

    expect(blockedAttack({ attacker, target }).target).toEqual(target);
  });
});

describe("describeAttack", () => {
  const attacker = { name: "Bolshack Dragon", power: 7000 };

  it("says who attacks what, with both powers when known", () => {
    expect(
      describeAttack(
        blockedAttack({
          attacker,
          target: { name: "Aqua Hulcus", power: 2000 },
        }),
      ),
    ).toBe(
      "Bolshack Dragon with 7000 power attacks Aqua Hulcus with 2000 power",
    );
    expect(describeAttack(blockedAttack({ attacker }))).toBe(
      "Bolshack Dragon with 7000 power attacks You",
    );
  });
});
