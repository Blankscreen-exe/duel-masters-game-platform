import { describe, expect, it } from "vitest";
import { answerMessage } from "./answers";

describe("answering a question", () => {
  it("names the question answered", () => {
    expect(answerMessage(7, { cards: ["a", "b"], cancel: false })).toEqual({
      header: "action",
      prompt: 7,
      cards: ["a", "b"],
      cancel: false,
    });
  });

  it("keeps a count and a cancel", () => {
    expect(answerMessage(2, { cards: [], cancel: true, count: 3 })).toEqual({
      header: "action",
      prompt: 2,
      cards: [],
      cancel: true,
      count: 3,
    });
  });
});
