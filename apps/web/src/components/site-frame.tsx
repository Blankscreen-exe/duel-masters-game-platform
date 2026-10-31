import type { ReactNode } from 'react';
import { SiteFooter } from './site-footer';
import { SiteHeader } from './site-header';
import { TabBar } from './tab-bar';
import { VerifyEmailNotice } from './verify-email-notice';

/**
 * Every page but a game's: the header, the page in the middle, the footer and the phone tab bar.
 *
 * A page marked with FILL_SCREEN (lib/fill-screen.ts) is, on a wide screen, exactly as tall as the
 * screen leaves between the header and the footer: it lays out its own parts in that height,
 * scrolling inside them, so the whole page is seen without scrolling. A screen shorter than 36rem
 * scrolls a little rather than squeezing it further. On a phone it stacks and scrolls as any page
 * does.
 */
export function SiteFrame({ children }: { readonly children: ReactNode }) {
  return (
    <>
      <SiteHeader />
      <VerifyEmailNotice />
      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:px-6 lg:has-[[data-fill-screen]]:flex lg:has-[[data-fill-screen]]:min-h-0 lg:has-[[data-fill-screen]]:flex-col">
        {children}
      </main>
      <SiteFooter />
      <TabBar />
    </>
  );
}
