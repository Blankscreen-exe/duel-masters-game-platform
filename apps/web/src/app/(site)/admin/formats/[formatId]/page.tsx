import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { FormatEditor } from '@/components/admin/format-editor';
import { serverApi } from '@/lib/api/server';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Format · Admin' };

export default async function FormatPage({ params }: PageProps<'/admin/formats/[formatId]'>) {
  const { formatId } = await params;
  await requireAdmin(`/admin/formats/${encodeURIComponent(formatId)}`);
  const api = await serverApi();
  const [format, filters] = await Promise.all([
    api.GET('/admin/formats/{formatId}', { params: { path: { formatId } } }),
    api.GET('/cards/filters'),
  ]);

  if (format.response.status === 404) notFound();
  if (!format.data || !filters.data) throw new Error('The format editor could not be loaded');

  // Keyed by format, so moving to another starts the editor afresh.
  return <FormatEditor key={format.data.id} format={format.data} filters={filters.data} />;
}
