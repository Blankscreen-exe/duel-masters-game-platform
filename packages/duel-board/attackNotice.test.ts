import { describe, expect, it } from "vitest";
import {
  attackToConfirm,
  battleOutcome,
  blockedAttack,
  describeAttack,
  finishingAttack,
} from "./attackNotice";
import { card, matchState, shield } from "./test/fixtures";
import {
  TAPPED_FLAG,
  type ActionMessage,
  type CreatureStatus,
  type MatchState,
} from "./types";

function battle({
  attackerPower = 3000,
  targetPower = 2000,
  attackerStatuses = [],
  targetStatuses = [],
}: {
  attackerPower?: number;
  targetPower?: number;
  attackerStatuses?: CreatureStatus[];
  targetStatuses?: CreatureStatus[];
} = {}): MatchState {
  return matchState(
    {
      playzone: [card("a")],
      powers: {
        a: { printed: 1000, current: 1000, attacking: attackerPower },
      },
      statuses: { a: attackerStatuses },
    },
    {
      playzone: [card("t", TAPPED_FLAG)],
      powers: {
        t: { printed: targetPower, current: targetPower, attacking: 9000 },
      },
      statuses: { t: targetStatuses },
      shieldzone: [shield("s1"), shield("s2"), shield("s3")],
      shieldMap: { s1: 1, s2: 2, s3: 3 },
    },
  );
}

describe("battleOutcome", () => {
  it("compares the power it attacks with against the target's power now", () => {
    expect(
      battleOutcome(
        battle({ attackerPower: 3000, targetPower: 2000 }),
        "a",
        "t",
      ),
    ).toMatchObject({
      tone: "win",
      headline: "You win the battle",
      destroyed: ["Card t"],
    });
  });

  it("destroys the attacker when the target has more power", () => {
    expect(
      battleOutcome(
        battle({ attackerPower: 2000, targetPower: 3000 }),
        "a",
        "t",
      ),
    ).toMatchObject({
      tone: "lose",
      headline: "You lose the battle",
      destroyed: ["Card a"],
    });
  });

  it("destroys both in a tie", () => {
    expect(
      battleOutcome(
        battle({ attackerPower: 2000, targetPower: 2000 }),
        "a",
        "t",
      ),
    ).toMatchObject({
      tone: "draw",
      destroyed: ["Card a", "Card t"],
    });
  });

  it("takes the winner down with a loser that has slayer", () => {
    expect(
      battleOutcome(battle({ targetStatuses: [{ id: "slayer" }] }), "a", "t")
        ?.destroyed,
    ).toEqual(["Card a", "Card t"]);
    expect(
      battleOutcome(
        battle({
          attackerPower: 1000,
          targetPower: 2000,
          attackerStatuses: [{ id: "slayer" }],
        }),
        "a",
        "t",
      )?.destroyed,
    ).toEqual(["Card a", "Card t"]);
  });

  it("destroys a winner that does not survive winning", () => {
    expect(
      battleOutcome(
        battle({ attackerStatuses: [{ id: "destroy_after_battle" }] }),
        "a",
        "t",
      )?.destroyed,
    ).toEqual(["Card a", "Card t"]);
    expect(
      battleOutcome(
        battle({
          attackerPower: 1000,
          targetPower: 2000,
          targetStatuses: [{ id: "destroy_after_battle" }],
        }),
        "a",
        "t",
      )?.destroyed,
    ).toEqual(["Card a", "Card t"]);
  });

  it("is null when either creature has left", () => {
    expect(battleOutcome(battle(), "gone", "t")).toBeNull();
    expect(battleOutcome(battle(), "a", "gone")).toBeNull();
  });

  describe("blockers", () => {
    const withBlocker = (
      flags = 0,
      attackerStatuses: CreatureStatus[] = [],
    ) => {
      const state = battle({ attackerStatuses });
      state.opponent.playzone.push(card("b", flags));
      state.opponent.statuses.b = [{ id: "blocker" }];
      return state;
    };

    it("names an untapped blocker", () => {
      expect(battleOutcome(withBlocker(), "a", "t")?.blockers).toEqual([
        "Card b",
      ]);
    });

    it("leaves out a tapped one", () => {
      expect(
        battleOutcome(withBlocker(TAPPED_FLAG), "a", "t")?.blockers,
      ).toEqual([]);
    });

    it("has none for a creature that can't be blocked", () => {
      expect(
        battleOutcome(withBlocker(0, [{ id: "cant_be_blocked" }]), "a", "t")
          ?.blockers,
      ).toEqual([]);
    });
  });
});

describe("attackToConfirm", () => {
  const question = (overrides: Partial<ActionMessage>): ActionMessage => ({
    header: "action",
    actionType: "" as ActionMessage["actionType"],
    text: "",
    cancellable: true,
    source: { virtualId: "a", uid: "", name: "Card a" },
    ...overrides,
  });

  it("compares both creatures once the target is chosen", () => {
    expect(
      attackToConfirm(battle(), question({ kind: "attack_target" }), ["t"]),
    ).toMatchObject({
      side: "mine",
      attacker: { name: "Card a", power: 3000 },
      target: { name: "Card t", power: 2000 },
      outcome: { tone: "win", destroyed: ["Card t"] },
    });
  });

  it("names the shield once it is chosen", () => {
    expect(
      attackToConfirm(
        battle(),
        question({ kind: "break_shields", maxSelections: 1 }),
        ["s2"],
      ),
    ).toMatchObject({
      target: { name: "Shield 2" },
      outcome: { headline: "Breaks shield 2", destroyed: [] },
    });
  });

  it("names every shield a double breaker breaks, in order", () => {
    expect(
      attackToConfirm(
        battle(),
        question({ kind: "break_shields", maxSelections: 2 }),
        ["s3", "s1"],
      )?.target.name,
    ).toBe("Shields 1 and 3");
  });

  it("leaves an incomplete selection for the server to refuse", () => {
    expect(
      attackToConfirm(
        battle(),
        question({ kind: "break_shields", maxSelections: 2 }),
        ["s1"],
      ),
    ).toBeNull();
    expect(
      attackToConfirm(battle(), question({ kind: "attack_target" }), []),
    ).toBeNull();
  });

  it("doesn't ask once the attack can't be called off", () => {
    expect(
      attackToConfirm(
        battle(),
        question({
          kind: "break_shields",
          maxSelections: 1,
          cancellable: false,
        }),
        ["s1"],
      ),
    ).toBeNull();
  });

  it("ignores any other question", () => {
    expect(
      attackToConfirm(battle(), question({ kind: "effect" }), ["t"]),
    ).toBeNull();
  });
});

describe("finishingAttack", () => {
  it("is for an opponent with no shields left", () => {
    const state = battle();
    state.opponent.shieldzone = [];

    expect(finishingAttack(state, "a")).toMatchObject({
      attacker: { name: "Card a", power: 3000 },
      target: { name: "Opponent" },
      outcome: { tone: "win", headline: "You win the game" },
    });
  });

  it("waits for the shield to be chosen while they have shields", () => {
    expect(finishingAttack(battle(), "a")).toBeNull();
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
