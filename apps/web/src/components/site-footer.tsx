/** The bottom of every page: credits and the fan project notice (design/SPEC.md, section 4.4). */
export function SiteFooter() {
  return (
    <footer className="mt-16 border-t border-slate-800 px-4 py-8 text-xs text-slate-500 sm:px-6">
      <div className="mx-auto max-w-7xl space-y-2">
        <p>
          A non-commercial fan project, not affiliated with, endorsed, sponsored or approved by
          Wizards of the Coast or Takara Tomy. Duel Masters card pictures and text are © Wizards of
          the Coast and Takara Tomy.
        </p>
        <p>
          Games are played with the rules engine of the open-source{' '}
          <a
            href="https://github.com/sindreslungaard/duel-masters"
            className="text-slate-400 underline-offset-4 hover:text-white hover:underline"
          >
            duel-masters
          </a>{' '}
          project (MIT license).
        </p>
      </div>
    </footer>
  );
}
