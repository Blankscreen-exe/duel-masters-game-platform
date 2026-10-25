import { SiteFrame } from '@/components/site-frame';

export default function SiteLayout({ children }: LayoutProps<'/'>) {
  return <SiteFrame>{children}</SiteFrame>;
}
