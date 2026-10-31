import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const BOARD_FOLDER = fileURLToPath(new URL(".", import.meta.url));

describe("card pictures", () => {
  it("all come from the site using the board, never from a fixed address", () => {
    const components = readdirSync(BOARD_FOLDER).filter((name) =>
      name.endsWith(".tsx"),
    );
    expect(components).toContain("Duel.tsx");

    for (const name of components) {
      const source = readFileSync(BOARD_FOLDER + name, "utf8");
      // The one address allowed is the name of the SVG format, in icons.
      expect(source, name).not.toMatch(/https?:\/\/(?!www\.w3\.org)/);
    }
  });
});
