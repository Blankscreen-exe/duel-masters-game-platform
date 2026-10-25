import Link from 'next/link';
import { buttonClasses } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';
import { SiteFrame } from '@/components/site-frame';

// Above both the site and the game pages, so it brings the site's frame itself.
export default function NotFound() {
  return (
    <SiteFrame>
      <Panel accent="darkness" className="mx-auto max-w-md p-6 text-center">
        <h1 className="font-display text-3xl font-bold tracking-widest text-white uppercase">
          Not found
        </h1>
        <p className="mt-3 text-slate-400">There is no page here.</p>
        <Link href="/" className={buttonClasses('secondary', 'mt-6')}>
          Back home
        </Link>
      </Panel>
    </SiteFrame>
  );
}
