import { useEffect, useState } from "react";

/** Below this the fixed 300px side column leaves too little room for the board,
 * so the chat moves into a drawer and the controls into a bar along the bottom.
 * Covers large phones in landscape and tablets in portrait, while laptops from
 * 1280 up keep the side column. Kept in step with the min-[1200px] classes that
 * size the floating overlays. */
export const COMPACT_VIEWPORT_QUERY = "(max-width: 1199px)";

/** A phone on its side, too short to give up height to a bar along the bottom. */
export const SHORT_LANDSCAPE_QUERY =
  "(orientation: landscape) and (max-height: 500px)";

/** A phone held upright, where the board is far taller than it is wide. */
export const PHONE_PORTRAIT_QUERY =
  "(orientation: portrait) and (max-width: 767px)";

/** A screen worked with a finger rather than a mouse. */
export const TOUCH_QUERY = "(pointer: coarse)";

/** Whether the page currently matches a media query, kept up to date. */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() =>
    typeof window === "undefined" ? false : window.matchMedia(query).matches,
  );

  useEffect(() => {
    const media = window.matchMedia(query);
    const update = () => setMatches(media.matches);

    update();
    media.addEventListener("change", update);

    return () => media.removeEventListener("change", update);
  }, [query]);

  return matches;
}
