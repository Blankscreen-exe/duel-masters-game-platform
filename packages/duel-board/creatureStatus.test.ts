import { describe, expect, it } from "vitest";
import {
  boardStatusBadges,
  findCreatureStatuses,
  MAX_BOARD_BADGES,
  statusBadges,
} from "./creatureStatus";
import { matchState } from "./test/fixtures";

describe("statusBadges", () => {
  it("describes a power attacker's bonus", () => {
    expect(statusBadges([{ id: "power_attacker", value: 3000 }])).toEqual([
      {
        id: "power_attacker",
        icon: "plus",
        tone: "attack",
        label: "Power attacker: +3000 power while attacking",
        granted: false,
      },
    ]);
  });

  it("counts the extra shields a creature breaks", () => {
    expect(
      statusBadges([
        { id: "shieldbreak_modifier", value: 1 },
        { id: "shieldbreak_modifier", value: 2 },
      ]).map((badge) => badge.label),
    ).toEqual(["Breaks 1 more shield", "Breaks 2 more shields"]);
  });

  it("gives the two attack limits their own icons", () => {
    const [creatures, players] = statusBadges([
      { id: "cant_attack_creatures" },
      { id: "cant_attack_players" },
    ]);

    expect(creatures.icon).not.toBe(players.icon);
  });

  it("colours each status by its kind", () => {
    expect(
      statusBadges([
        { id: "summoning_sickness" },
        { id: "slayer" },
        { id: "blocker" },
        { id: "survivor" },
        { id: "doesnt_untap" },
      ]).map((badge) => badge.tone),
    ).toEqual(["sickness", "attack", "defense", "support", "limit"]);
  });

  it("marks a status another card gave", () => {
    const [badge] = statusBadges([{ id: "slayer", granted: true }]);

    expect(badge.granted).toBe(true);
  });

  it("keeps the server's order and leaves out what it does not know", () => {
    expect(
      statusBadges([
        { id: "blocker" },
        { id: "a_status_from_the_future" },
        { id: "slayer" },
      ]).map((badge) => badge.id),
    ).toEqual(["blocker", "slayer"]);
  });

  it("shows nothing without statuses", () => {
    expect(statusBadges()).toEqual([]);
  });
});

describe("boardStatusBadges", () => {
  const ids = (statuses: { id: string; granted?: boolean }[]) =>
    boardStatusBadges(statuses).shown.map((badge) => badge.id);

  it("shows each status that matters at a glance", () => {
    for (const id of [
      "blocker",
      "slayer",
      "cant_be_blocked",
      "stealth",
      "cant_be_attacked",
      "attack_untapped",
      "treated_as_tapped",
      "power_attacker",
      "cant_attack_creatures",
      "cant_attack_players",
      "survivor",
      "summoning_sickness",
    ]) {
      expect(ids([{ id }])).toEqual([id]);
    }
  });

  it("leaves the rest to viewing the card", () => {
    expect(
      boardStatusBadges([
        { id: "doublebreaker" },
        { id: "triplebreaker" },
        { id: "shieldbreak_modifier", value: 1 },
        { id: "speed_attacker" },
        { id: "cant_be_selected_by_opp" },
        { id: "cant_be_tapped_by_opp" },
        { id: "doesnt_untap" },
        { id: "destroy_after_battle" },
        { id: "tap_ability" },
        { id: "silent_skill" },
        { id: "wave_striker" },
      ]),
    ).toEqual({ shown: [], more: 0 });
  });

  it("puts the most important first", () => {
    expect(
      ids([
        { id: "summoning_sickness" },
        { id: "power_attacker" },
        { id: "slayer" },
        { id: "blocker" },
      ]),
    ).toEqual(["blocker", "slayer", "power_attacker"]);
  });

  it("counts what does not fit instead of showing it", () => {
    const badges = boardStatusBadges([
      { id: "summoning_sickness" },
      { id: "survivor" },
      { id: "power_attacker" },
      { id: "slayer" },
      { id: "blocker" },
      { id: "doublebreaker" },
    ]);

    expect(badges.shown).toHaveLength(MAX_BOARD_BADGES);
    // Double breaker is never shown on the board, so it is not counted.
    expect(badges.more).toBe(2);
  });

  it("shows can't be blocked once alongside stealth, granted if either is", () => {
    const badges = boardStatusBadges([
      { id: "cant_be_blocked" },
      { id: "stealth", granted: true },
    ]);

    expect(badges).toMatchObject({
      shown: [{ icon: "unblockable", granted: true }],
      more: 0,
    });
  });
});

describe("findCreatureStatuses", () => {
  const state = {
    ...matchState(),
    me: { ...matchState().me, statuses: { mine: [{ id: "blocker" }] } },
    opponent: {
      ...matchState().opponent,
      statuses: { theirs: [{ id: "slayer" }] },
    },
  };

  it("finds a creature on either side", () => {
    expect(findCreatureStatuses(state, "mine")).toEqual([{ id: "blocker" }]);
    expect(findCreatureStatuses(state, "theirs")).toEqual([{ id: "slayer" }]);
  });

  it("is empty for a card that is not a creature in play", () => {
    expect(findCreatureStatuses(state, "elsewhere")).toEqual([]);
  });
});
