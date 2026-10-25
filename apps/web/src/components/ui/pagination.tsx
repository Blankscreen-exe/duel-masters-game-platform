import Link from 'next/link';
import { buttonClasses } from './button';

interface PaginationProps {
  readonly page: number;
  readonly pages: number;
  /** The address of another page. */
  readonly href: (page: number) => string;
}

/** Previous and next links with where the reader is; nothing when everything fits on one page. */
export function Pagination({ page, pages, href }: PaginationProps) {
  if (pages <= 1) return null;

  return (
    <nav aria-label="Pages" className="flex items-center justify-center gap-3">
      <PageLink page={page - 1} pages={pages} href={href} label="Previous" />
      <span className="font-display text-sm font-semibold tracking-wider text-slate-400 uppercase">
        Page {page} of {pages}
      </span>
      <PageLink page={page + 1} pages={pages} href={href} label="Next" />
    </nav>
  );
}

function PageLink({
  page,
  pages,
  href,
  label,
}: Pick<PaginationProps, 'pages' | 'href'> & { readonly page: number; readonly label: string }) {
  if (page < 1 || page > pages) {
    return (
      <span aria-disabled className={buttonClasses('secondary', 'pointer-events-none opacity-40')}>
        {label}
      </span>
    );
  }

  return (
    <Link
      href={href(page)}
      rel={label === 'Next' ? 'next' : 'prev'}
      className={buttonClasses('secondary')}
    >
      {label}
    </Link>
  );
}
