import { createContext, useContext, useMemo, type CSSProperties } from "react";

/** Where card pictures come from, which the site using the board decides. */
export interface CardPictures {
  /** The picture of a card, by its engine id. */
  card(imageId: string): string;
  /** The back of a card, shown face down and while pictures load. */
  back: string;
}

/** What the server sends as the picture of a card the player may not see. */
export const FACE_DOWN_IMAGE_ID = "backside";

/** A card's picture, or the card back when it is face down or unknown. */
export function cardPictureUrl(
  pictures: CardPictures,
  imageId?: string | null,
): string {
  return imageId && imageId !== FACE_DOWN_IMAGE_ID
    ? pictures.card(imageId)
    : pictures.back;
}

const CardPicturesContext = createContext<CardPictures | null>(null);

/** Gives the board, and the previews used beside it, their card pictures. */
export const CardPicturesProvider = CardPicturesContext.Provider;

/** Width over height of a card picture: they are about 0.72, the shape of a
 * real card. */
export const CARD_ASPECT_RATIO = 403 / 560;

/** The card pictures of the board, with the style that keeps a picture the
 * shape of a card, showing the card back until it has loaded. Without the
 * shape an image sized by its height alone is zero pixels wide while it
 * loads, so cards would jump sideways as they arrive and could not be
 * measured for animation. */
export function useCardPictures(): {
  url: (imageId?: string | null) => string;
  placeholder: CSSProperties;
} {
  const pictures = useContext(CardPicturesContext);
  if (!pictures) {
    throw new Error("The duel board needs a CardPicturesProvider for its card pictures");
  }

  return useMemo(
    () => ({
      url: (imageId?: string | null) => cardPictureUrl(pictures, imageId),
      placeholder: {
        aspectRatio: `auto ${CARD_ASPECT_RATIO}`,
        backgroundImage: `url(${pictures.back})`,
        backgroundSize: "cover",
      },
    }),
    [pictures],
  );
}

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
