'use client';

import { useEffect, useState } from 'react';
import { api, errorMessage, UNREACHABLE } from './api/client';
import { hasMoreCards, withCardPage, type CardPages } from './card-pages';
import type { CardSearch } from './card-search';

interface CardPagesState {
  /** The cards held, which may still be the last search's while a new one loads. */
  readonly pages: CardPages | null;
  /** Whether this search's first page is still on its way. */
  readonly searching: boolean;
  /** Why the last page asked for did not come. */
  readonly error: string | null;
  /** Whether this search has more cards to ask for. */
  readonly more: boolean;
  /** Asks for the next page, or again for one that failed. */
  readonly loadMore: () => void;
}

/** A card search read a page at a time, each page joining the cards already held. */
export function useCardPages(
  search: Omit<CardSearch, 'page' | 'pageSize'>,
  pageSize: number,
): CardPagesState {
  const key = JSON.stringify(search);
  const [wanted, setWanted] = useState({ key, page: 1, attempt: 0 });
  const [pages, setPages] = useState<CardPages | null>(null);
  const [failed, setFailed] = useState<{ asked: string; message: string } | null>(null);

  // A changed search starts again from its first page, even one searched for before.
  if (wanted.key !== key) setWanted({ key, page: 1, attempt: 0 });
  const page = wanted.key === key ? wanted.page : 1;
  const attempt = wanted.key === key ? wanted.attempt : 0;
  // Names this very request, so a failure is forgotten once it is asked for again.
  const asked = JSON.stringify([key, page, attempt]);

  useEffect(() => {
    const query: CardSearch = { ...(JSON.parse(key) as CardSearch), page, pageSize };
    const controller = new AbortController();
    api
      .GET('/cards', { params: { query }, signal: controller.signal })
      .then(({ data, error }) => {
        if (data) setPages((held) => withCardPage(held, key, data));
        else setFailed({ asked, message: errorMessage(error) });
      })
      .catch(() => {
        // A search replaced by a newer one is cancelled on purpose.
        if (!controller.signal.aborted) setFailed({ asked, message: UNREACHABLE });
      });
    return () => {
      controller.abort();
    };
  }, [key, page, pageSize, asked]);

  const current = pages?.key === key ? pages : null;
  const error = failed?.asked === asked ? failed.message : null;
  const loading = error === null && (current === null || current.page < page);
  const searching = error === null && current === null;
  const more = error !== null || (current !== null && !loading && hasMoreCards(current));
  const heldPage = current?.page ?? 0;

  function loadMore() {
    setWanted((now) =>
      now.page > heldPage
        ? { ...now, attempt: now.attempt + 1 }
        : { ...now, page: heldPage + 1, attempt: 0 },
    );
  }

  return { pages, searching, error, more, loadMore };
}
