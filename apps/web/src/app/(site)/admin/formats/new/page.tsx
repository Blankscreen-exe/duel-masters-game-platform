import type { Metadata } from 'next';
import { FormatEditor } from '@/components/admin/format-editor';
import { serverApi } from '@/lib/api/server';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'New format · Admin' };

export default async function NewFormatPage() {
  await requireAdmin('/admin/formats/new');
  const api = await serverApi();
  const { data: filters } = await api.GET('/cards/filters');

  if (!filters) throw new Error('The format editor could not be loaded');

  return <FormatEditor format={null} filters={filters} />;
}
