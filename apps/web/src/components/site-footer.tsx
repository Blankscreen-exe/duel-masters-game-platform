/**
 * The bottom of every page: the fan project notice and credits (design/SPEC.md, section 4.4), in
 * one thin line on a wide screen, so that pages that fit the screen have room for it.
 */
export function SiteFooter() {
  return (
    <footer className="border-t border-slate-800 px-4 py-2.5 text-xs text-slate-500 sm:px-6">
      <p className="mx-auto max-w-7xl">
        Non-commercial fan project, not affiliated with or endorsed by Wizards of the Coast or
        Takara Tomy, who own the card pictures and text. Rules engine:{' '}
        <a
          href="https://github.com/sindreslungaard/duel-masters"
          className="text-slate-400 underline-offset-4 hover:text-white hover:underline"
        >
          duel-masters
        </a>{' '}
        (MIT license).
      </p>
    </footer>
  );
}
