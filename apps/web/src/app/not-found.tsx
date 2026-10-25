import Link from 'next/link';
import { buttonClasses } from '@/components/ui/button';
import { Panel } from '@/components/ui/panel';

export default function NotFound() {
  return (
    <Panel accent="darkness" className="mx-auto max-w-md p-6 text-center">
      <h1 className="font-display text-3xl font-bold tracking-widest text-white uppercase">
        Not found
      </h1>
      <p className="mt-3 text-slate-400">There is no page here.</p>
      <Link href="/" className={buttonClasses('secondary', 'mt-6')}>
        Back home
      </Link>
    </Panel>
  );
}
