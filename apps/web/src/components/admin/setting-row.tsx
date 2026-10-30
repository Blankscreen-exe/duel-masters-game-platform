'use client';

import { useState, type SubmitEvent } from 'react';
import { Button } from '@/components/ui/button';
import { FIELD_CONTROL } from '@/components/ui/field';
import { settingLimits, type AdminSetting } from '@/lib/admin';
import { resetSetting, setSetting } from '@/lib/admin-actions';
import type { Outcome } from '@/lib/api/outcome';
import { notify } from '@/lib/notifications';

/**
 * One server setting: what it does, its value, and a way to change it or put it back to its
 * default. The server checks a value against the setting's limits and applies it at once.
 */
export function SettingRow({ setting: initial }: { readonly setting: AdminSetting }) {
  const [setting, setShown] = useState(initial);
  const [value, setValue] = useState(initial.value);
  const [pending, setPending] = useState(false);
  const limits = settingLimits(setting);

  async function run(send: () => Promise<Outcome<AdminSetting>>, done: string) {
    setPending(true);
    const outcome = await send();
    setPending(false);
    if (!outcome.ok) {
      notify(outcome.message);
      return;
    }
    setShown(outcome.value);
    setValue(outcome.value.value);
    notify(done);
  }

  function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    void run(() => setSetting(setting.key, value), 'Setting saved');
  }

  return (
    <li className="grid gap-3 border-b border-slate-800/80 px-4 py-4 last:border-0 md:grid-cols-[minmax(0,1fr)_22rem] md:items-center md:gap-6">
      <div className="min-w-0">
        <p className="text-sm text-slate-100">{setting.description}</p>
        <p className="mt-1 text-xs break-words text-slate-500">
          {setting.key} · Default {setting.default}
          {limits && ` · ${limits}`}
        </p>
      </div>

      <form onSubmit={submit} className="flex items-center gap-2">
        <div className="min-w-0 flex-1">
          {setting.kind === 'boolean' ? (
            <label className="flex min-h-11 cursor-pointer items-center gap-2 text-sm text-slate-300 hover:text-white">
              <input
                type="checkbox"
                checked={value === 'true'}
                onChange={(event) => {
                  setValue(String(event.target.checked));
                }}
                className="size-4 accent-sky-400"
              />
              {value === 'true' ? 'Yes' : 'No'}
            </label>
          ) : (
            <input
              aria-label={setting.key}
              type={setting.kind === 'number' ? 'number' : 'text'}
              value={value}
              min={setting.min}
              max={setting.max}
              maxLength={setting.maxLength}
              onChange={(event) => {
                setValue(event.target.value);
              }}
              className={FIELD_CONTROL}
            />
          )}
        </div>
        <Button type="submit" disabled={pending || value === setting.value}>
          Save
        </Button>
        <Button
          variant="secondary"
          disabled={pending || setting.isDefault}
          onClick={() => void run(() => resetSetting(setting.key), 'Setting put back')}
        >
          Default
        </Button>
      </form>
    </li>
  );
}
