/** A game fills the whole screen, without the site's header, footer and tab bar. */
export default function GameLayout({ children }: LayoutProps<'/'>) {
  return <main className="flex min-h-dvh flex-1 flex-col">{children}</main>;
}
