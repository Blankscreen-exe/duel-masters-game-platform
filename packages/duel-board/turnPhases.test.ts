import { describe, expect, it } from "vitest";
import {
  attackForfeit,
  describeAttackForfeit,
  phaseBanners,
  phaseOfStep,
  queueBanners,
  TURN_PHASES,
  turnHint,
  turnNumber,
  bannerHoldSeconds,
  bannerSeconds,
  displayPhase,
  type PhaseBanner,
  type TurnPhase,
} from "./turnPhases";
import { PLAYABLE_FLAG, type MatchState, type TurnStep } from "./types";
import { card, matchState } from "./test/fixtures";

describe("phaseOfStep", () => {
  it.each<[TurnStep, string]>([
    ["begin_turn", "draw"],
    ["untap", "draw"],
    ["start_of_turn", "draw"],
    ["draw", "draw"],
    ["charge", "mana"],
    ["main", "summon"],
    ["attack", "attack"],
    ["end", "end"],
    ["end_of_turn", "end"],
  ])("shows the %s step as the %s phase", (step, phase) => {
    expect(phaseOfStep(step)).toBe(phase);
  });

  it("has no phase before the first turn", () => {
    expect(phaseOfStep("")).toBeNull();
  });

  it("has no phase for a step it does not know", () => {
    expect(phaseOfStep("overtime" as TurnStep)).toBeNull();
  });

  it("lists the phases in turn order", () => {
    expect(TURN_PHASES.map((info) => info.phase)).toEqual([
      "draw",
      "mana",
      "summon",
      "attack",
      "end",
    ]);
  });
});

describe("turnNumber", () => {
  it("is the turn the duel is on", () => {
    expect(turnNumber(matchState({}, {}, { turn: 7 }))).toBe(7);
  });

  it("is missing before the first turn has begun", () => {
    expect(turnNumber(matchState({}, {}, { turn: 0, step: "" }))).toBeNull();
  });

  it("is missing when the server does not count turns", () => {
    const { turn: _turn, ...withoutTurn } = matchState();

    expect(turnNumber(withoutTurn as MatchState)).toBeNull();
  });
});

describe("turnHint", () => {
  it("invites the player to add mana at the start of their turn", () => {
    expect(turnHint(matchState())).toBe(
      "Add a card to your mana zone, then summon or attack",
    );
  });

  it("moves on to summoning once mana was added", () => {
    expect(
      turnHint(
        matchState(
          {},
          {},
          { hasAddedManaThisRound: true, turnManaCharged: true },
        ),
      ),
    ).toBe("Summon, cast spells or attack. Adding mana is over for this turn");
  });

  it("says mana is over once a card was played", () => {
    expect(
      turnHint(
        matchState({}, {}, { step: "main", canChargeManaThisRound: false }),
      ),
    ).toBe("Summon, cast spells or attack. Adding mana is over for this turn");
  });

  it("says summoning is over in the attack step", () => {
    expect(
      turnHint(
        matchState({}, {}, { step: "attack", hasAttackedThisRound: true }),
      ),
    ).toBe(
      "Attack or end your turn. Summoning and mana are over for this turn",
    );
  });

  it("names the opponent's turn", () => {
    expect(turnHint(matchState({}, {}, { myTurn: false }))).toBe(
      "Opponent's turn",
    );
  });

  it("says nothing to a spectator", () => {
    expect(
      turnHint(matchState({}, {}, { myTurn: false, spectator: true })),
    ).toBeNull();
  });

  it("says nothing before the first turn", () => {
    expect(turnHint(matchState({}, {}, { step: "" }))).toBeNull();
  });
});

describe("attackForfeit", () => {
  const playable = (id: string) => card(id, PLAYABLE_FLAG);

  it("counts the cards the player could still play and the mana still to add", () => {
    const state = matchState({
      hand: [playable("a"), playable("b"), card("c")],
    });

    expect(attackForfeit(state)).toEqual({
      playableCards: 2,
      canAddMana: true,
    });
  });

  it("warns about mana alone when nothing is affordable", () => {
    const state = matchState({ hand: [card("a")] });

    expect(attackForfeit(state)).toEqual({
      playableCards: 0,
      canAddMana: true,
    });
  });

  it("warns about playable cards alone once mana was added", () => {
    const state = matchState(
      { hand: [playable("a")] },
      {},
      { hasAddedManaThisRound: true },
    );

    expect(attackForfeit(state)).toEqual({
      playableCards: 1,
      canAddMana: false,
    });
  });

  it("has nothing to lose with an empty hand", () => {
    expect(attackForfeit(matchState())).toBeNull();
  });

  it("has nothing to lose once mana was added and nothing is affordable", () => {
    const state = matchState(
      { hand: [card("a")] },
      {},
      { step: "main", canChargeManaThisRound: false },
    );

    expect(attackForfeit(state)).toBeNull();
  });

  it("has nothing to lose once already attacking", () => {
    const state = matchState(
      { hand: [playable("a")] },
      {},
      { step: "attack", hasAttackedThisRound: true },
    );

    expect(attackForfeit(state)).toBeNull();
  });

  it("does not apply on the opponent's turn or to spectators", () => {
    const hand = { hand: [playable("a")] };

    expect(attackForfeit(matchState(hand, {}, { myTurn: false }))).toBeNull();
    expect(attackForfeit(matchState(hand, {}, { spectator: true }))).toBeNull();
  });
});

describe("describeAttackForfeit", () => {
  it("names everything still possible", () => {
    expect(describeAttackForfeit({ playableCards: 2, canAddMana: true })).toBe(
      "You can still play 2 cards from your hand and add mana. Once you attack or use a tap ability, you can't summon, cast spells or add mana for the rest of this turn.",
    );
  });

  it("uses the singular for one card", () => {
    expect(
      describeAttackForfeit({ playableCards: 1, canAddMana: false }),
    ).toMatch(/^You can still play 1 card from your hand\. /);
  });

  it("mentions only mana when no card is affordable", () => {
    expect(
      describeAttackForfeit({ playableCards: 0, canAddMana: true }),
    ).toMatch(/^You can still add mana\. /);
  });
});

describe("phaseBanners", () => {
  it("announces nothing for the first state of a connection", () => {
    expect(phaseBanners(null, matchState())).toEqual([]);
  });

  it("announces the player's turn and its phase", () => {
    const previous = matchState({}, {}, { myTurn: false, step: "main" });
    const next = matchState({}, {}, { myTurn: true, step: "begin_turn" });

    expect(phaseBanners(previous, next)).toEqual([
      { kind: "turn", title: "Your turn", tone: "mine" },
      { kind: "phase", phase: "draw", title: "Draw phase", tone: "mine" },
    ]);
  });

  it("announces the opponent's turn", () => {
    const previous = matchState({}, {}, { myTurn: true, step: "attack" });
    const next = matchState({}, {}, { myTurn: false, step: "draw" });

    expect(phaseBanners(previous, next)).toEqual([
      { kind: "turn", title: "Opponent's turn", tone: "theirs" },
      { kind: "phase", phase: "draw", title: "Draw phase", tone: "theirs" },
    ]);
  });

  it("announces each phase within a turn", () => {
    const draw = matchState({}, {}, { step: "draw" });
    const mana = matchState({}, {}, { step: "charge" });
    const summon = matchState({}, {}, { step: "main" });
    const attack = matchState({}, {}, { step: "attack" });

    expect(phaseBanners(draw, mana)).toEqual([
      { kind: "phase", phase: "mana", title: "Mana phase", tone: "mine" },
    ]);
    expect(phaseBanners(mana, summon)).toEqual([
      { kind: "phase", phase: "summon", title: "Summon phase", tone: "mine" },
    ]);
    expect(phaseBanners(summon, attack)).toEqual([
      { kind: "phase", phase: "attack", title: "Attack phase", tone: "mine" },
    ]);
  });

  it("stays quiet while the phase does not change", () => {
    const state = matchState({}, {}, { step: "main" });

    expect(phaseBanners(state, { ...state })).toEqual([]);
  });

  it("treats steps of one phase as the same phase", () => {
    const untap = matchState({}, {}, { step: "untap" });
    const draw = matchState({}, {}, { step: "draw" });

    expect(phaseBanners(untap, draw)).toEqual([]);
  });

  it("announces a new turn to a spectator by the turn starting over", () => {
    const watching = { myTurn: false, spectator: true };
    const previous = matchState({}, {}, { ...watching, step: "attack" });
    const next = matchState({}, {}, { ...watching, step: "begin_turn" });

    expect(phaseBanners(previous, next)).toEqual([
      { kind: "turn", title: "Next turn", tone: "neutral" },
      { kind: "phase", phase: "draw", title: "Draw phase", tone: "neutral" },
    ]);
  });

  it("tells a turn from its start when the board only saw it later", () => {
    const previous = matchState({}, {}, { myTurn: false, step: "attack" });
    const next = matchState({}, {}, { myTurn: true, step: "charge" });

    expect(phaseBanners(previous, next).map((banner) => banner.title)).toEqual([
      "Your turn",
      "Draw phase",
      "Mana phase",
    ]);
  });

  it("tells a spectator a new turn from its phase going back", () => {
    const watching = { myTurn: false, spectator: true };
    const previous = matchState({}, {}, { ...watching, step: "attack" });
    const next = matchState({}, {}, { ...watching, step: "charge" });

    expect(phaseBanners(previous, next).map((banner) => banner.title)).toEqual([
      "Next turn",
      "Draw phase",
      "Mana phase",
    ]);
  });

  it("announces an extra turn for the same player", () => {
    const previous = matchState({}, {}, { step: "attack" });
    const next = matchState({}, {}, { step: "begin_turn" });

    expect(phaseBanners(previous, next).map((banner) => banner.kind)).toEqual([
      "turn",
      "phase",
    ]);
  });
});

describe("queueBanners", () => {
  const turn = (title: string): PhaseBanner => ({
    kind: "turn",
    title,
    tone: "mine",
  });
  const phase = (title: string, phase?: TurnPhase): PhaseBanner => ({
    kind: "phase",
    phase,
    title,
    tone: "mine",
  });

  it("shows the first title at once and queues the rest", () => {
    expect(queueBanners([], [turn("Your turn"), phase("Draw phase")])).toEqual([
      turn("Your turn"),
      phase("Draw phase"),
    ]);
  });

  it("replaces a phase still waiting with a newer one", () => {
    const queue = [turn("Your turn"), phase("Draw phase")];

    expect(queueBanners(queue, [phase("Mana phase")])).toEqual([
      turn("Your turn"),
      phase("Mana phase"),
    ]);
  });

  it("never takes the title on screen away", () => {
    expect(
      queueBanners([phase("Mana phase")], [phase("Summon phase")]),
    ).toEqual([phase("Mana phase"), phase("Summon phase")]);
  });

  it("replaces everything, the title on screen too, when a turn starts", () => {
    const queue = [phase("Summon phase"), phase("Attack phase")];
    const next = [turn("Opponent's turn"), phase("Draw phase", "draw")];

    expect(queueBanners(queue, next)).toEqual(next);
  });

  it("keeps the draw waiting, so a turn is told to begin in order", () => {
    const queue = [turn("Your turn"), phase("Draw phase", "draw")];

    expect(queueBanners(queue, [phase("Mana phase", "mana")])).toEqual([
      turn("Your turn"),
      phase("Draw phase", "draw"),
      phase("Mana phase", "mana"),
    ]);
  });

  it("leaves the queue alone when nothing new arrives", () => {
    const queue = [phase("Mana phase")];

    expect(queueBanners(queue, [])).toEqual(queue);
  });
});

describe("displayPhase", () => {
  it("shows the mana phase until mana is in", () => {
    expect(displayPhase(matchState({}, {}, { step: "charge" }))).toBe("mana");
  });

  it("shows the summon phase once mana is in, on either player's turn", () => {
    expect(
      displayPhase(
        matchState({}, {}, { step: "charge", turnManaCharged: true }),
      ),
    ).toBe("summon");
    expect(
      displayPhase(
        matchState(
          {},
          {},
          { myTurn: false, step: "charge", turnManaCharged: true },
        ),
      ),
    ).toBe("summon");
  });

  it("follows the server's step otherwise", () => {
    expect(
      displayPhase(
        matchState({}, {}, { step: "attack", turnManaCharged: true }),
      ),
    ).toBe("attack");
  });

  it("announces the summon phase as soon as mana is in", () => {
    const before = matchState({}, {}, { step: "charge" });
    const after = matchState({}, {}, { step: "charge", turnManaCharged: true });

    expect(phaseBanners(before, after)).toEqual([
      { kind: "phase", phase: "summon", title: "Summon phase", tone: "mine" },
    ]);
  });
});

describe("bannerSeconds", () => {
  it("keeps the start of a turn brief", () => {
    const turn: PhaseBanner = {
      kind: "turn",
      title: "Your turn",
      tone: "mine",
    };
    const draw: PhaseBanner = {
      kind: "phase",
      phase: "draw",
      title: "Draw phase",
      tone: "mine",
    };
    const mana: PhaseBanner = {
      kind: "phase",
      phase: "mana",
      title: "Mana phase",
      tone: "mine",
    };

    expect(bannerHoldSeconds(turn)).toBe(0.8);
    expect(bannerHoldSeconds(draw)).toBe(0.6);
    expect(bannerHoldSeconds(mana)).toBe(1);
    expect(bannerSeconds(mana)).toBeCloseTo(1.7);
  });
});
