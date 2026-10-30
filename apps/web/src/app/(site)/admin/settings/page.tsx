import type { Metadata } from 'next';
import { SettingRow } from '@/components/admin/setting-row';
import { Panel } from '@/components/ui/panel';
import { groupSettings } from '@/lib/admin';
import { serverApi } from '@/lib/api/server';
import { requireAdmin } from '@/lib/session';

export const metadata: Metadata = { title: 'Settings · Admin' };

/** Every server setting, by group. Changes apply at once, without a restart. */
export default async function AdminSettingsPage() {
  await requireAdmin('/admin/settings');
  const api = await serverApi();
  const { data: settings, response } = await api.GET('/admin/settings');

  if (!settings) {
    throw new Error(`The settings failed to load with HTTP ${String(response.status)}`);
  }

  return (
    <div className="space-y-6">
      <h1 className="font-display text-4xl font-bold tracking-widest text-white uppercase">
        Settings
      </h1>
      {groupSettings(settings).map(({ group, settings: inGroup }) => (
        <Panel key={group} title={group}>
          <ul>
            {inGroup.map((setting) => (
              <SettingRow key={setting.key} setting={setting} />
            ))}
          </ul>
        </Panel>
      ))}
    </div>
  );
}
