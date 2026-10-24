import { describe, expect, it } from "vitest";
import { diffBoard } from "./boardDiff";
import type { BoardMeasurements } from "./cardPositions";
import { planEchoes } from "./echoes";
import { flightDuration, planFlights, type BoardChange } from "./flights";
import type { MatchState } from "./types";
import { card, matchState, measuredCard, measurements } from "./test/fixtures";

function change(
  previous: MatchState,
  next: MatchState,
  before: BoardMeasurements,
  after: BoardMeasurements,
): BoardChange {
  return { previous, next, moves: diffBoard(previous, next), before, after };
}

describe("planEchoes", () => {
  it("echoes a card that acted in place at once", () => {
    const state = matchState({ playzone: [card("a")] });
    const screen = measurements([measuredCard("a", 100, 300)]);
    const update = change(
      state,
      { ...state, activatedCards: ["a"] },
      screen,
      screen,
    );

    expect(planEchoes(update, planFlights(update))).toEqual([
      expect.objectContaining({ virtualId: "a", imageId: "image-a", delay: 0 }),
    ]);
  });

  it("waits for a summoned card to land before echoing it", () => {
    const previous = matchState({ hand: [card("a")] });
    const next = matchState(
      { playzone: [card("a")] },
      {},
      { activatedCards: ["a"] },
    );
    const update = change(
      previous,
      next,
      measurements([measuredCard("a", 100, 600)]),
      measurements([measuredCard("a", 100, 300)]),
    );
    const flights = planFlights(update);
    const [echo] = planEchoes(update, flights);

    expect(echo.box).toMatchObject({ x: 100, y: 300 });
    expect(echo.delay).toBe(flightDuration(flights[0]));
  });

  it("echoes a card that left the screen where it was last seen", () => {
    const previous = matchState({ playzone: [card("a")] });
    const next = matchState({}, {}, { activatedCards: ["a"] });
    const update = change(
      previous,
      next,
      measurements([measuredCard("a", 100, 300)]),
      measurements(),
    );

    expect(planEchoes(update, planFlights(update))).toEqual([
      expect.objectContaining({
        box: expect.objectContaining({ x: 100, y: 300 }),
        delay: 0,
      }),
    ]);
  });

  it("skips a card that cannot be seen", () => {
    const state = matchState();
    const update = change(
      state,
      { ...state, activatedCards: ["hidden", "scrolled"] },
      measurements([measuredCard("scrolled", 2000, 300, { visible: false })]),
      measurements([measuredCard("scrolled", 2000, 300, { visible: false })]),
    );

    expect(planEchoes(update, [])).toEqual([]);
  });

  it("echoes nothing when nothing acted", () => {
    const state = matchState({ playzone: [card("a")] });
    const screen = measurements([measuredCard("a", 100, 300)]);

    expect(planEchoes(change(state, state, screen, screen), [])).toEqual([]);
  });
});
