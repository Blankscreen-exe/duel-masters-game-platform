/** A player's answer to a question the engine asked them. */
export interface ActionAnswer {
  cards: string[];
  cancel: boolean;
  count?: number;
}

/**
 * The message answering question `prompt`. The engine numbers every question
 * and only takes an answer naming the one still open, so a late or repeated
 * click can never answer the next question instead.
 */
export function answerMessage(prompt: number, answer: ActionAnswer) {
  return { header: "action", prompt, ...answer };
}
