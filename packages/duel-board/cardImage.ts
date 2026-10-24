/** The scan of a card, or the card back when the card is face down or its
 * image is unknown. */
export function cardImageUrl(imageId?: string | null): string {
  return `https://scans.shobu.io/${imageId || "backside"}.jpg`;
}

/** Width over height of a card scan, which are 403 by 560 pixels. */
export const CARD_ASPECT_RATIO = 403 / 560;

/** Styles a card image so that, until its scan has loaded, it keeps the shape
 * of a card and shows the card back. Without the shape an image sized by its
 * height alone is zero pixels wide while it loads, so cards would jump sideways
 * as they arrive and could not be measured for animation. */
export const CARD_IMAGE_PLACEHOLDER = {
  aspectRatio: `auto ${CARD_ASPECT_RATIO}`,
  backgroundImage: `url(${cardImageUrl()})`,
  backgroundSize: "cover",
} as const;

/** How a used up card is drawn: without colour and darkened. Written out in
 * full, alongside its plain counterpart, so an animation can blend the two. */
export const MUTED_CARD_FILTER = "grayscale(1) brightness(0.55)";
export const PLAIN_CARD_FILTER = "grayscale(0) brightness(1)";

/** A deep blue glass over a shield, with a streak of light across it and a
 * glow around its edge, like the shields of the original game. */
export const SHIELD_TINT = {
  borderRadius: "5%",
  background:
    "linear-gradient(135deg, rgba(37, 99, 235, 0.6) 0%, rgba(30, 64, 175, 0.55) 38%, rgba(191, 219, 254, 0.6) 48%, rgba(30, 64, 175, 0.5) 58%, rgba(23, 37, 84, 0.65) 100%)",
  boxShadow:
    "inset 0 0 0 1px rgba(147, 197, 253, 0.8), inset 0 0 12px rgba(37, 99, 235, 0.65), 0 0 8px rgba(37, 99, 235, 0.6)",
} as const;
