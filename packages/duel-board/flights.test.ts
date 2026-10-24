import { describe, expect, it } from "vitest";
import { diffBoard } from "./boardDiff";
import type { BoardMeasurements, ScreenBox } from "./cardPositions";
import {
  anchorId,
  flightDuration,
  holdTurnDraws,
  planFlights,
  rotationDelta,
  type BoardChange,
  type Flight,
} from "./flights";
import type { MatchState } from "./types";
import {
  box,
  card,
  matchState,
  measuredCard,
  measurements,
  shield,
} from "./test/fixtures";

function change(
  previous: MatchState,
  next: MatchState,
  before: BoardMeasurements,
  after: BoardMeasurements,
): BoardChange {
  return { previous, next, moves: diffBoard(previous, next), before, after };
}

describe("planFlights", () => {
  it("flies nothing when nothing moved", () => {
    const state = matchState({ playzone: [card("a")] });
    const screen = measurements([measuredCard("a", 100, 300)]);

    expect(planFlights(change(state, state, screen, screen))).toEqual([]);
  });

  it("slides a card that moved along its row", () => {
    const previous = matchState({ manazone: [card("a"), card("b")] });
    const next = matchState({ manazone: [card("b"), card("a")] });

    const flights = planFlights(
      change(
        previous,
        next,
        measurements([
          measuredCard("a", 100, 300),
          measuredCard("b", 180, 300),
        ]),
        measurements([
          measuredCard("b", 100, 300),
          measuredCard("a", 180, 300),
        ]),
      ),
    );

    expect(flights).toEqual([
      {
        virtualId: "a",
        imageId: "image-a",
        from: box(100, 300),
        to: box(180, 300),
        landsOnCard: true,
        fade: null,
        muted: { from: false, to: false },
        delay: 0,
      },
      {
        virtualId: "b",
        imageId: "image-b",
        from: box(180, 300),
        to: box(100, 300),
        landsOnCard: true,
        fade: null,
        muted: { from: false, to: false },
        delay: 0,
      },
    ]);
  });

  it("slides the cards left behind when one leaves the row", () => {
    const previous = matchState({ playzone: [card("a"), card("b")] });
    const next = matchState({ playzone: [card("b")], graveyard: [card("a")] });

    const flights = planFlights(
      change(
        previous,
        next,
        measurements([
          measuredCard("a", 100, 300),
          measuredCard("b", 180, 300),
        ]),
        measurements([
          measuredCard("b", 100, 300),
          measuredCard("a", 900, 600),
        ]),
      ),
    );

    expect(flights).toEqual([
      expect.objectContaining({ virtualId: "a", to: box(900, 600) }),
      expect.objectContaining({ virtualId: "b", to: box(100, 300) }),
    ]);
  });

  it("turns a card tapped in place", () => {
    const previous = matchState({ playzone: [card("a")] });
    const next = matchState({ playzone: [card("a", 1)] });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements([measuredCard("a", 100, 300)]),
        measurements([measuredCard("a", 100, 300, { rotation: 90 })]),
      ),
    );

    expect(flight.from.rotation).toBe(0);
    expect(flight.to.rotation).toBe(90);
    expect(flight.landsOnCard).toBe(true);
  });

  it("greys out mana as it is tapped", () => {
    const previous = matchState({ manazone: [card("a")] });
    const next = matchState({ manazone: [card("a", 1)] });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements([measuredCard("a", 100, 500)]),
        measurements([
          measuredCard("a", 100, 500, { rotation: 90, muted: true }),
        ]),
      ),
    );

    expect(flight.muted).toEqual({ from: false, to: true });
  });

  it("brings the colour back as mana untaps", () => {
    const previous = matchState({ manazone: [card("a", 1)] });
    const next = matchState({ manazone: [card("a")] });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements([
          measuredCard("a", 100, 500, { rotation: 90, muted: true }),
        ]),
        measurements([measuredCard("a", 100, 500)]),
      ),
    );

    expect(flight.muted).toEqual({ from: true, to: false });
  });

  it("animates a card that only changed colour", () => {
    const state = matchState({ manazone: [card("a")] });

    const flights = planFlights(
      change(
        state,
        state,
        measurements([measuredCard("a", 100, 500)]),
        measurements([measuredCard("a", 100, 500, { muted: true })]),
      ),
    );

    expect(flights).toHaveLength(1);
  });

  it("keeps a greyed out card grey while it flies to the graveyard pile", () => {
    const previous = matchState({ manazone: [card("a", 1)] });
    const next = matchState({ graveyard: [card("a")] });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements([
          measuredCard("a", 100, 500, { rotation: 90, muted: true }),
        ]),
        measurements([], [[anchorId("me", "graveyard"), box(900, 600)]]),
      ),
    );

    expect(flight.muted).toEqual({ from: true, to: true });
  });

  it("flies a summoned card from the hand to the battle zone", () => {
    const previous = matchState({ hand: [card("a")] });
    const next = matchState({ playzone: [card("a")] });

    const flights = planFlights(
      change(
        previous,
        next,
        measurements([measuredCard("a", 100, 600)]),
        measurements([measuredCard("a", 100, 300)]),
      ),
    );

    expect(flights).toEqual([
      expect.objectContaining({
        from: box(100, 600),
        to: box(100, 300),
        landsOnCard: true,
        fade: null,
      }),
    ]);
  });

  it("draws a card from the deck pile into the hand", () => {
    const previous = matchState({}, {});
    const next = matchState({ hand: [card("a")], deck: 29 });
    const deck = box(900, 500, { width: 40, height: 56 });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements([], [[anchorId("me", "deck"), deck]]),
        measurements([measuredCard("a", 100, 600)]),
      ),
    );

    // The copy starts card shaped, as tall as the pile.
    expect(flight.from).toEqual(box(900, 500, { width: 40, height: 56 }));
    expect(flight.to).toEqual(box(100, 600));
    expect(flight.landsOnCard).toBe(true);
    expect(flight.fade).toBeNull();
  });

  it("fades a drawn card in when the deck pile is folded away", () => {
    const previous = matchState();
    const next = matchState({ hand: [card("a")] });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements(),
        measurements([measuredCard("a", 100, 600)]),
      ),
    );

    expect(flight.from).toEqual(box(100, 600));
    expect(flight.to).toEqual(box(100, 600));
    expect(flight.fade).toBe("in");
  });

  it("fades in rather than flying from an anchor scrolled out of view", () => {
    const previous = matchState();
    const next = matchState({ hand: [card("a")] });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements(
          [],
          [[anchorId("me", "deck"), box(900, 500, { visible: false })]],
        ),
        measurements([measuredCard("a", 100, 600)]),
      ),
    );

    expect(flight.fade).toBe("in");
  });

  it("flies a card the opponent plays from their hidden hand", () => {
    const previous = matchState({}, { handCount: 5 });
    const next = matchState({}, { handCount: 4, playzone: [card("a")] });
    const hand = box(900, 50, { height: 20 });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements(
          [],
          [
            [anchorId("opponent", "hand"), hand],
            [anchorId("opponent", "deck"), box(900, 150)],
          ],
        ),
        measurements([measuredCard("a", 100, 200, { rotation: 180 })]),
      ),
    );

    expect(flight.from).toMatchObject({
      x: 900,
      y: 50,
      height: 20,
      rotation: 180,
    });
  });

  it("flies an opponent's card put straight from their deck from the deck pile", () => {
    const previous = matchState({}, { handCount: 5 });
    const next = matchState({}, { handCount: 5, playzone: [card("a")] });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements(
          [],
          [
            [anchorId("opponent", "hand"), box(900, 50)],
            [anchorId("opponent", "deck"), box(900, 150)],
          ],
        ),
        measurements([measuredCard("a", 100, 200)]),
      ),
    );

    expect(flight.from).toMatchObject({ x: 900, y: 150 });
  });

  it("flies a card into the hidden hand of a player a spectator watches", () => {
    const previous = matchState(
      { playzone: [card("a")], handCount: 3 },
      {},
      { spectator: true },
    );
    const next = matchState({ handCount: 4 }, {}, { spectator: true });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements([measuredCard("a", 100, 300)]),
        measurements([], [[anchorId("me", "hand"), box(900, 700)]]),
      ),
    );

    expect(flight.to).toMatchObject({ x: 900, y: 700 });
    expect(flight.landsOnCard).toBe(false);
  });

  it("flies a destroyed card onto the graveyard pile below the top card", () => {
    const previous = matchState({
      playzone: [card("a")],
      graveyard: [card("top")],
    });
    const next = matchState({ graveyard: [card("a"), card("top")] });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements(
          [measuredCard("a", 100, 300), measuredCard("top", 900, 600)],
          [[anchorId("me", "graveyard"), box(900, 600)]],
        ),
        measurements(
          [measuredCard("top", 900, 600)],
          [[anchorId("me", "graveyard"), box(900, 600)]],
        ),
      ),
    );

    expect(flight).toMatchObject({
      virtualId: "a",
      to: { x: 900, y: 600 },
      landsOnCard: false,
      fade: null,
    });
  });

  it("flies a card to its zone's row when it lands scrolled out of view", () => {
    const previous = matchState({ hand: [card("a")] });
    const next = matchState({ manazone: [card("a")] });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements(
          [measuredCard("a", 100, 600)],
          [[anchorId("me", "manazone"), box(400, 500)]],
        ),
        measurements(
          [measuredCard("a", 1400, 500, { visible: false, rotation: 180 })],
          [[anchorId("me", "manazone"), box(400, 500)]],
        ),
      ),
    );

    expect(flight.to).toMatchObject({ x: 400, y: 500, rotation: 180 });
    expect(flight.landsOnCard).toBe(false);
  });

  it("fades out a card that went somewhere with no place on screen", () => {
    const previous = matchState(
      {},
      { shieldzone: [shield("s")], handCount: 2 },
    );
    const next = matchState({}, { handCount: 3 });

    const [flight] = planFlights(
      change(
        previous,
        next,
        measurements([measuredCard("s", 300, 100)]),
        measurements(),
      ),
    );

    expect(flight).toMatchObject({
      from: box(300, 100),
      to: box(300, 100),
      landsOnCard: false,
      fade: "out",
    });
  });

  it("uses the image the card shows once it lands", () => {
    const previous = matchState({ shieldzone: [shield("s")] });
    const next = matchState({ hand: [card("s")] });
    const before = measurements([measuredCard("s", 300, 450)]);
    before.cards.get("s")!.imageId = "backside";

    const [flight] = planFlights(
      change(
        previous,
        next,
        before,
        measurements([measuredCard("s", 100, 600)]),
      ),
    );

    expect(flight.imageId).toBe("image-s");
  });

  it("leaves alone a card that was only scrolled into view", () => {
    const state = matchState({ manazone: [card("a")] });

    const flights = planFlights(
      change(
        state,
        state,
        measurements([measuredCard("a", 1400, 500, { visible: false })]),
        measurements([measuredCard("a", 400, 500)]),
      ),
    );

    expect(flights).toEqual([]);
  });

  it("leaves alone a graveyard card covered by a newer one", () => {
    const previous = matchState({
      playzone: [card("a")],
      graveyard: [card("old")],
    });
    const next = matchState({ graveyard: [card("old"), card("a")] });

    const flights = planFlights(
      change(
        previous,
        next,
        measurements([
          measuredCard("a", 100, 300),
          measuredCard("old", 900, 600),
        ]),
        measurements([measuredCard("a", 900, 600)]),
      ),
    );

    expect(flights.map((flight) => flight.virtualId)).toEqual(["a"]);
  });
});

describe("holdTurnDraws", () => {
  const deck = box(900, 500, { width: 40, height: 56 });

  /** A card drawn between two states, the first from the opponent's turn. */
  function draw(
    previousStep: MatchState["step"],
    nextStep: MatchState["step"],
    previousTurnWasMine = false,
  ) {
    const previous = matchState(
      {},
      {},
      { step: previousStep, myTurn: previousTurnWasMine },
    );
    const next = matchState(
      { hand: [card("drawn")], playzone: [card("other")] },
      {},
      { step: nextStep },
    );
    const update = change(
      { ...previous, me: { ...previous.me, playzone: [card("other")] } },
      next,
      measurements(
        [measuredCard("other", 100, 300)],
        [[anchorId("me", "deck"), deck]],
      ),
      measurements([
        measuredCard("drawn", 100, 600),
        measuredCard("other", 160, 300),
      ]),
    );

    return holdTurnDraws(planFlights(update), update, 1.5);
  }

  it("holds back the card drawn as the turn starts", () => {
    const flights = draw("attack", "draw");

    expect(flights.find((flight) => flight.virtualId === "drawn")?.delay).toBe(
      1.5,
    );
  });

  it("holds it back too when the board first sees the turn past its draw", () => {
    const flights = draw("attack", "charge");

    expect(flights.find((flight) => flight.virtualId === "drawn")?.delay).toBe(
      1.5,
    );
  });

  it("leaves every other card alone", () => {
    const flights = draw("attack", "draw");

    expect(flights.find((flight) => flight.virtualId === "other")?.delay).toBe(
      0,
    );
  });

  it("does not hold back a card drawn later by an effect", () => {
    const flights = draw("main", "main", true);

    expect(flights.every((flight) => flight.delay === 0)).toBe(true);
  });
});

describe("flightDuration", () => {
  function flightBetween(
    from: ScreenBox,
    to: ScreenBox,
    fade: Flight["fade"] = null,
  ): Flight {
    return {
      virtualId: "a",
      imageId: "a",
      from,
      to,
      landsOnCard: true,
      fade,
      muted: { from: false, to: false },
      delay: 0,
    };
  }

  it("turns a card in place quickly", () => {
    expect(
      flightDuration(flightBetween(box(0, 0), box(0, 0, { rotation: 90 }))),
    ).toBe(0.3);
  });

  it("takes longer the further the card travels", () => {
    const short = flightDuration(flightBetween(box(0, 0), box(100, 0)));
    const long = flightDuration(flightBetween(box(0, 0), box(0, 500)));

    expect(short).toBeGreaterThan(0.3);
    expect(long).toBeGreaterThan(short);
  });

  it("caps a journey across the whole screen", () => {
    expect(flightDuration(flightBetween(box(0, 0), box(3000, 2000)))).toBe(0.6);
  });

  it("fades in a fixed time", () => {
    expect(flightDuration(flightBetween(box(0, 0), box(0, 0), "in"))).toBe(
      0.25,
    );
  });
});

describe("rotationDelta", () => {
  it.each([
    [0, 0, 0],
    [0, 90, 90],
    [90, 0, -90],
    [270, 0, 90],
    [180, -90, 90],
    [-90, 90, -180],
  ])("turns from %i to %i by %i degrees", (from, to, delta) => {
    expect(rotationDelta(from, to)).toBe(delta);
  });
});
