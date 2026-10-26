/**
 * Marks a page that fills the screen on a wide one, between the header and the footer (see
 * SiteFrame): spread it on the page's outermost element. The page then lays out its own parts in
 * that height and scrolls inside them.
 */
export const FILL_SCREEN = { 'data-fill-screen': true } as const;
