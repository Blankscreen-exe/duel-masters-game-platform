import type { ReactNode } from 'react';
import { SiteFooter } from './site-footer';
import { SiteHeader } from './site-header';
import { TabBar } from './tab-bar';

/** Every page but a game's: the header, the page in the middle, the footer and the phone tab bar. */
export function SiteFrame({ children }: { readonly children: ReactNode }) {
  return (
    <>
      <SiteHeader />
      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:px-6">{children}</main>
      <SiteFooter />
      <TabBar />
    </>
  );
}
