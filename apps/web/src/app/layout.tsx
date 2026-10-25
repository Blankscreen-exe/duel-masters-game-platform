import type { Metadata, Viewport } from 'next';
import localFont from 'next/font/local';
import { Suspense } from 'react';
import { LiveProvider } from '@/components/live/live-provider';
import { LiveSession } from '@/components/live/live-session';
import { Toaster } from '@/components/ui/toaster';
import './globals.css';

// Self-hosted (SIL Open Font License, see ./fonts), so the site never depends on reaching Google.
const rajdhani = localFont({
  src: [
    { path: './fonts/rajdhani-latin-500-normal.woff2', weight: '500' },
    { path: './fonts/rajdhani-latin-600-normal.woff2', weight: '600' },
    { path: './fonts/rajdhani-latin-700-normal.woff2', weight: '700' },
  ],
  variable: '--font-rajdhani',
  display: 'swap',
});
const inter = localFont({
  src: './fonts/inter-latin-wght-normal.woff2',
  weight: '100 900',
  variable: '--font-inter',
  display: 'swap',
});

export const metadata: Metadata = {
  title: { default: 'Duel Masters', template: '%s · Duel Masters' },
  description: 'Play Duel Masters online: build decks from every card and duel other players.',
};

export const viewport: Viewport = {
  themeColor: '#07090e',
};

export default function RootLayout({ children }: LayoutProps<'/'>) {
  return (
    <html lang="en" className={`${rajdhani.variable} ${inter.variable}`}>
      <body className="flex flex-col">
        <LiveProvider>
          {children}
          <Suspense>
            <LiveSession />
          </Suspense>
        </LiveProvider>
        <Toaster />
      </body>
    </html>
  );
}
