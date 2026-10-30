import type { Metadata } from 'next';
import { OpponentEditor } from '@/components/admin/opponent-editor';
import { serverApi } from '@/lib/api/server';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'New computer opponent · Admin' };

export default async function NewComputerOpponentPage() {
  await requireAdmin('/admin/computer-opponents/new');
  const api = await serverApi();
  const [options, formats, filters] = await Promise.all([
    api.GET('/admin/computer-opponents/options'),
    api.GET('/formats'),
    api.GET('/cards/filters'),
  ]);

  if (!options.data || !formats.data || !filters.data) {
    throw new Error('The computer opponent editor could not be loaded');
  }

  return (
    <OpponentEditor
      opponent={null}
      options={options.data}
      formats={formats.data}
      filters={filters.data}
    />
  );
}
