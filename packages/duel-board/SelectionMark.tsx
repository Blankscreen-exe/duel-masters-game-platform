import { MotionConfig, motion } from "motion/react";

/** The outline of a card picked in a selection. */
export const SELECTED_CARD_OUTLINE = "ring-4 ring-green-500";

/** Marks a picked card with a green badge in its middle: a tick, or the card's
 * place in the order when the cards are being ordered. Place it inside the
 * card's relatively positioned wrapper. */
export function SelectionMark({ order }: { order?: number }) {
  return (
    <MotionConfig reducedMotion="user">
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
        <motion.div
          className="flex aspect-square w-[35%] max-w-[4rem] items-center justify-center rounded-full bg-green-500 text-white shadow-[0_0_10px_rgba(0,0,0,0.6)]"
          initial={{ scale: 0.5, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ duration: 0.15, ease: "easeOut" }}
        >
          {order === undefined ? (
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              className="h-3/5 w-3/5"
              fill="none"
              stroke="currentColor"
              strokeWidth={3}
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M5 12.5l4.5 4.5L19 7.5" />
            </svg>
          ) : (
            <span className="text-2xl font-bold leading-none">{order}</span>
          )}
        </motion.div>
      </div>
    </MotionConfig>
  );
}
