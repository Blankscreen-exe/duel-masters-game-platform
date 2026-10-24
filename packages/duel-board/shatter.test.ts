import { describe, expect, it } from "vitest";
import { shatter } from "./shatter";

const shatterShards = (seed: string) => shatter(seed).shards;

/** The corners of a shard, back out of its clip-path, in percent. */
function corners(clipPath: string): [number, number][] {
  return [...clipPath.matchAll(/(-?[\d.]+)% (-?[\d.]+)%/g)].map((match) => [
    Number(match[1]),
    Number(match[2]),
  ]);
}

function triangleArea([a, b, c]: [number, number][]): number {
  return (
    Math.abs((b[0] - a[0]) * (c[1] - a[1]) - (c[0] - a[0]) * (b[1] - a[1])) / 2
  );
}

describe("shatterShards", () => {
  it("breaks a shield into triangles that cover all of it", () => {
    const shards = shatterShards("shield-1");
    const area = shards.reduce(
      (sum, shard) => sum + triangleArea(corners(shard.clipPath)),
      0,
    );

    expect(shards).toHaveLength(12);
    shards.forEach((shard) => expect(corners(shard.clipPath)).toHaveLength(3));
    expect(area).toBeCloseTo(100 * 100, 0);
  });

  it("sends every piece away from the point of impact", () => {
    for (const shard of shatterShards("shield-2")) {
      const [impact] = corners(shard.clipPath);
      const outward =
        (shard.centre.x * 100 - impact[0]) * shard.dx +
        (shard.centre.y * 100 - impact[1]) * shard.dy;

      expect(outward).toBeGreaterThan(0);
    }
  });

  it("keeps every piece's flight and turn within bounds", () => {
    for (const shard of shatterShards("shield-3")) {
      const flight = Math.hypot(shard.dx, shard.dy);

      expect(flight).toBeGreaterThanOrEqual(0.45);
      expect(flight).toBeLessThanOrEqual(0.95);
      expect(Math.abs(shard.rotate)).toBeGreaterThanOrEqual(60);
      expect(Math.abs(shard.rotate)).toBeLessThanOrEqual(260);
      expect(shard.delay).toBeGreaterThanOrEqual(0);
      expect(shard.delay).toBeLessThanOrEqual(0.05);
    }
  });

  it("breaks the same shield the same way every time", () => {
    expect(shatterShards("shield-4")).toEqual(shatterShards("shield-4"));
  });

  it("strikes near the middle of the shield", () => {
    const { impact, shards } = shatter("shield-7");

    expect(impact.x).toBeGreaterThanOrEqual(0.38);
    expect(impact.x).toBeLessThanOrEqual(0.62);
    expect(impact.y).toBeGreaterThanOrEqual(0.33);
    expect(impact.y).toBeLessThanOrEqual(0.57);
    // Every crack starts where the shield was struck.
    for (const shard of shards) {
      const [x, y] = corners(shard.clipPath)[0];
      expect(x / 100).toBeCloseTo(impact.x, 2);
      expect(y / 100).toBeCloseTo(impact.y, 2);
    }
  });

  it("breaks different shields differently", () => {
    expect(shatterShards("shield-5")).not.toEqual(shatterShards("shield-6"));
  });
});
