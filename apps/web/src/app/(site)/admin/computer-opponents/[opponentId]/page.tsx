import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { OpponentEditor } from '@/components/admin/opponent-editor';
import { serverApi } from '@/lib/api/server';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Computer opponent · Admin' };

export default async function ComputerOpponentPage({
  params,
}: PageProps<'/admin/computer-opponents/[opponentId]'>) {
  const { opponentId } = await params;
  await requireAdmin(`/admin/computer-opponents/${encodeURIComponent(opponentId)}`);
  const api = await serverApi();
  const [opponent, options, formats, filters] = await Promise.all([
    api.GET('/admin/computer-opponents/{opponentId}', { params: { path: { opponentId } } }),
    api.GET('/admin/computer-opponents/options'),
    api.GET('/formats'),
    api.GET('/cards/filters'),
  ]);

  if (opponent.response.status === 404) notFound();
  if (!opponent.data || !options.data || !formats.data || !filters.data) {
    throw new Error('The computer opponent editor could not be loaded');
  }

  // Keyed by opponent, so moving to another starts the editor afresh.
  return (
    <OpponentEditor
      key={opponent.data.id}
      opponent={opponent.data}
      options={options.data}
      formats={formats.data}
      filters={filters.data}
    />
  );
}
