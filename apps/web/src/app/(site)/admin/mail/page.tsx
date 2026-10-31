import type { Metadata } from 'next';
import { MailSettingsForm } from '@/components/admin/mail-settings-form';
import { serverApi } from '@/lib/api/server';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Email · Admin' };

/** The mail server that sends new players their codes. */
export default async function AdminMailPage() {
  const admin = await requireAdmin('/admin/mail');
  const api = await serverApi();
  const { data: settings, response } = await api.GET('/admin/mail');

  if (!settings) {
    throw new Error(`The mail settings failed to load with HTTP ${String(response.status)}`);
  }

  return (
    <div className="space-y-6">
      <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
        Email
      </h1>
      <MailSettingsForm key={settings.updatedAt} settings={settings} testAddress={admin.email} />
    </div>
  );
}
