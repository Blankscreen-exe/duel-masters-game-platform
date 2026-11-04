import Link from 'next/link';

/**
 * The bottom of every page: the fan project notice (design/SPEC.md, section 4.4), in
 * one thin line on a wide screen, so that pages that fit the screen have room for it. It links to
 * How to play too, which the phone's tab bar has no room for.
 */
export function SiteFooter() {
  return (
    <footer className="border-t border-slate-800 px-4 py-2.5 text-xs text-slate-500 sm:px-6">
      <div className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-x-6 gap-y-1">
        <p>
          Non-commercial fan project, not affiliated with or endorsed by Wizards of the Coast or
          Takara Tomy, who own the card pictures and text.
        </p>
        <Link href="/how-to-play" className="font-semibold text-slate-400 hover:text-accent">
          How to play
        </Link>
      </div>
    </footer>
  );
}
