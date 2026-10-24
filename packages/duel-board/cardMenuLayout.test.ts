import { describe, expect, it } from "vitest";
import { CARD_ASPECT_RATIO } from "./cardImage";
import { liftedBox, restingTransform, sheetLayout } from "./cardMenuLayout";
import { box } from "./test/fixtures";

const desktop = { width: 1440, height: 900 };
const phoneUpright = { width: 390, height: 844 };
const phoneOnItsSide = { width: 844, height: 390 };

describe("liftedBox", () => {
  it("grows the card it came from, centred on it", () => {
    const lifted = liftedBox(
      box(700, 450, { height: 200, width: 144 }),
      desktop,
    );

    expect(lifted.height).toBe(300);
    expect(lifted.width).toBeCloseTo(300 * CARD_ASPECT_RATIO);
    expect(lifted).toMatchObject({ x: 700, y: 450, rotation: 0 });
  });

  it("is never too small for its buttons", () => {
    expect(liftedBox(box(700, 450, { height: 60 }), desktop).height).toBe(260);
  });

  it("stays on screen near an edge", () => {
    const lifted = liftedBox(box(5, 895, { height: 200 }), desktop);

    expect(lifted.x - lifted.width / 2).toBeGreaterThanOrEqual(12);
    expect(lifted.y + lifted.height / 2).toBeLessThanOrEqual(900 - 12);
  });

  it("goes to the middle of the screen without an origin", () => {
    expect(liftedBox(null, desktop)).toMatchObject({ x: 720, y: 450 });
  });
});

describe("sheetLayout", () => {
  const contents = { actions: 3, reasons: 1 };

  it("puts the sheet along the bottom of an upright phone", () => {
    const layout = sheetLayout(phoneUpright, contents);

    expect(layout.edge).toBe("bottom");
    expect(layout.size).toBe(16 * 2 + 20 + 28 + 3 * 56 + 18);
  });

  it("shows the card above the sheet, as large as fits", () => {
    const layout = sheetLayout(phoneUpright, contents);
    const { card } = layout;

    expect(card.y + card.height / 2).toBeLessThanOrEqual(844 - layout.size);
    expect(card.y - card.height / 2).toBeGreaterThanOrEqual(0);
    expect(card.width).toBeLessThanOrEqual(390 - 24);
    expect(card.x).toBe(195);
    expect(card.rotation).toBe(0);
  });

  it("never lets the sheet take most of the screen", () => {
    const many = sheetLayout(phoneUpright, {
      actions: 12,
      reasons: 12,
    });

    expect(many.size).toBeCloseTo(844 * 0.6);
  });

  it("puts the sheet on the right of a phone on its side", () => {
    const layout = sheetLayout(phoneOnItsSide, contents);

    expect(layout.edge).toBe("right");
    // 42% of the width would be wider than the sheet ever needs to be.
    expect(layout.size).toBe(340);
  });

  it("gives a narrow side sheet a share of the width", () => {
    const layout = sheetLayout({ width: 700, height: 360 }, contents);

    expect(layout.size).toBeCloseTo(700 * 0.42);
  });

  it("shows the card left of a side sheet, filling the height", () => {
    const layout = sheetLayout(phoneOnItsSide, contents);

    expect(layout.card.height).toBe(390 - 24);
    expect(layout.card.x + layout.card.width / 2).toBeLessThanOrEqual(
      844 - layout.size,
    );
  });
});

describe("restingTransform", () => {
  it("maps the shown card back onto the card in its row", () => {
    const origin = box(100, 700, { height: 100, rotation: 90 });
    const shown = box(195, 300, { height: 400 });

    expect(restingTransform(origin, shown)).toEqual({
      x: -95,
      y: 400,
      scale: 0.25,
      rotate: 90,
      opacity: 1,
    });
  });
});
