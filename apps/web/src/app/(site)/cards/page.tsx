import type { Metadata } from 'next';
import { CardFiltersForm } from '@/components/cards/card-filters-form';
import { CardTile } from '@/components/cards/card-tile';
import { Pagination } from '@/components/ui/pagination';
import { Panel } from '@/components/ui/panel';
import { serverApi } from '@/lib/api/server';
import { cardSearchHref, isFiltered, pageCount, parseCardSearch } from '@/lib/card-search';

export const metadata: Metadata = { title: 'Cards' };

export default async function CardsPage({ searchParams }: PageProps<'/cards'>) {
  const search = parseCardSearch(await searchParams);
  const api = await serverApi();
  const [results, filters] = await Promise.all([
    api.GET('/cards', { params: { query: search } }),
    api.GET('/cards/filters'),
  ]);

  if (!filters.data) throw new Error('The card filters could not be loaded');
  // A search the server refuses is shown as an empty result, with its reason.
  if (!results.data && results.response.status !== 400) {
    throw new Error(`The card search failed with HTTP ${String(results.response.status)}`);
  }

  const found = results.data;
  const pages = found ? pageCount(found.total, found.pageSize) : 1;
  const summary = results.data
    ? `${results.data.total.toLocaleString('en')} ${results.data.total === 1 ? 'card' : 'cards'}${isFiltered(search) ? ' found' : ''}`
    : results.error.message;

  return (
    <div className="grid gap-6 lg:grid-cols-[18rem_1fr]">
      <h1 className="sr-only">Cards</h1>
      <Panel title="Search" accent="water" className="self-start lg:sticky lg:top-20">
        <CardFiltersForm filters={filters.data} search={search} />
      </Panel>

      <div className="space-y-6">
        <p
          aria-live="polite"
          className="font-display text-sm font-semibold tracking-wider text-slate-400 uppercase"
        >
          {summary}
        </p>

        {found && found.cards.length > 0 ? (
          <ul className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 xl:grid-cols-5">
            {found.cards.map((card, index) => (
              <li key={card.id}>
                <CardTile card={card} priority={index < 10} />
              </li>
            ))}
          </ul>
        ) : (
          found && (
            <Panel className="p-8 text-center text-slate-400">
              No cards match. Try fewer filters or another word.
            </Panel>
          )
        )}

        {found && (
          <Pagination
            page={found.page}
            pages={pages}
            href={(page) => cardSearchHref(search, page)}
          />
        )}
      </div>
    </div>
  );
}
