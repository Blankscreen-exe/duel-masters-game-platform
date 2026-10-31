'use client';

import { useRouter } from 'next/navigation';
import { useState, type SubmitEvent } from 'react';
import { Button } from '@/components/ui/button';
import { Modal } from '@/components/ui/modal';
import { Panel } from '@/components/ui/panel';
import { SelectField } from '@/components/ui/select-field';
import { TextField } from '@/components/ui/text-field';
import { clearMailSettings, saveMailSettings, sendTestMail } from '@/lib/admin-actions';
import type { Outcome } from '@/lib/api/outcome';
import { textField } from '@/lib/form-data';
import {
  DEFAULT_MAIL_PORT,
  DEFAULT_MAIL_SECURITY,
  MAIL_SECURITIES,
  mailPassword,
  mailStatusText,
  type MailSecurity,
  type MailSettings,
} from '@/lib/mail-settings';
import { notify } from '@/lib/notifications';
import { EMAIL_MAX_LENGTH } from '@/lib/verification';

interface MailSettingsFormProps {
  readonly settings: MailSettings;
  /** The address offered for a test email: the admin's own. */
  readonly testAddress?: string;
}

/** Setting up the mail server, sending a test email through it, and forgetting it. */
export function MailSettingsForm({ settings, testAddress }: MailSettingsFormProps) {
  const router = useRouter();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [testError, setTestError] = useState<string | null>(null);
  const [confirmingClear, setConfirmingClear] = useState(false);
  const canStore = settings.keyAvailable;

  async function run<T>(
    send: () => Promise<Outcome<T>>,
    fail: (message: string) => void,
    done: () => void,
  ) {
    setPending(true);
    setError(null);
    setTestError(null);
    const outcome = await send();
    setPending(false);
    if (outcome.ok) done();
    else fail(outcome.message);
  }

  function save(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const password = mailPassword(textField(form, 'password'), form.has('removePassword'));
    const input = {
      host: textField(form, 'host'),
      port: Number(textField(form, 'port')),
      security: textField(form, 'security') as MailSecurity,
      username: textField(form, 'username'),
      fromAddress: textField(form, 'fromAddress'),
      fromName: textField(form, 'fromName'),
      ...(password === undefined ? {} : { password }),
    };

    void run(
      () => saveMailSettings(input),
      setError,
      () => {
        notify('Mail settings saved. Send a test email to check them.');
        router.refresh();
      },
    );
  }

  function test(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const to = textField(new FormData(event.currentTarget), 'to');

    void run(
      () => sendTestMail(to),
      setTestError,
      () => {
        notify(`The mail server took the test email for ${to}.`);
      },
    );
  }

  function clear() {
    void run(clearMailSettings, setError, () => {
      setConfirmingClear(false);
      notify('The mail server is forgotten.');
      router.refresh();
    });
  }

  return (
    <div className="space-y-6">
      <p
        role="status"
        className={`px-4 py-3 text-sm ring-1 ring-inset ${
          settings.configured
            ? 'bg-emerald-500/10 text-emerald-100 ring-emerald-400/30'
            : 'bg-amber-500/10 text-amber-100 ring-amber-400/30'
        }`}
      >
        {mailStatusText(settings)}
      </p>

      <Panel title="Mail server">
        <form onSubmit={save} className="grid gap-4 p-4 sm:grid-cols-2 sm:p-6">
          <TextField
            label="Server"
            name="host"
            required
            placeholder="smtp.example.com"
            defaultValue={settings.host}
            disabled={!canStore}
          />
          <TextField
            label="Port"
            name="port"
            type="number"
            required
            min={1}
            max={65535}
            defaultValue={settings.port ?? DEFAULT_MAIL_PORT}
            disabled={!canStore}
          />
          <SelectField
            label="Connection"
            name="security"
            className="sm:col-span-2"
            defaultValue={settings.security ?? DEFAULT_MAIL_SECURITY}
            disabled={!canStore}
          >
            {MAIL_SECURITIES.map(({ value, label }) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </SelectField>
          <TextField
            label="Username"
            name="username"
            autoComplete="off"
            defaultValue={settings.username}
            hint="Leave empty for a server that needs no login."
            disabled={!canStore}
          />
          <div>
            <TextField
              label="Password"
              name="password"
              type="password"
              autoComplete="new-password"
              placeholder={settings.hasPassword ? 'Stored; leave empty to keep it' : undefined}
              disabled={!canStore}
            />
            {settings.hasPassword && (
              <label className="mt-2 flex cursor-pointer items-center gap-2 text-sm text-slate-400">
                <input type="checkbox" name="removePassword" disabled={!canStore} />
                Remove the stored password
              </label>
            )}
          </div>
          <TextField
            label="Send from address"
            name="fromAddress"
            type="email"
            required
            maxLength={EMAIL_MAX_LENGTH}
            placeholder="noreply@example.com"
            defaultValue={settings.fromAddress}
            disabled={!canStore}
          />
          <TextField
            label="Send from name"
            name="fromName"
            maxLength={100}
            defaultValue={settings.fromName ?? 'Duel Masters'}
            disabled={!canStore}
          />

          {error && (
            <p
              role="alert"
              className="bg-red-500/10 px-3 py-2 text-sm text-red-200 ring-1 ring-red-400/40 ring-inset sm:col-span-2"
            >
              {error}
            </p>
          )}

          <div className="flex flex-wrap gap-3 sm:col-span-2">
            <Button type="submit" disabled={pending || !canStore}>
              Save
            </Button>
            {settings.configured && (
              <Button
                variant="secondary"
                disabled={pending}
                onClick={() => {
                  setConfirmingClear(true);
                }}
              >
                Forget this server
              </Button>
            )}
          </div>
        </form>
      </Panel>

      {settings.configured && (
        <Panel title="Send a test email">
          <form onSubmit={test} className="flex flex-wrap items-end gap-3 p-4 sm:px-6 sm:pt-6">
            <TextField
              label="Send to"
              name="to"
              type="email"
              required
              maxLength={EMAIL_MAX_LENGTH}
              defaultValue={testAddress}
              className="min-w-64 flex-1"
            />
            <Button type="submit" variant="secondary" disabled={pending}>
              {pending ? 'Sending…' : 'Send'}
            </Button>
          </form>
          {testError && (
            <div className="px-4 pb-4 sm:px-6 sm:pb-6">
              <p
                role="alert"
                className="bg-red-500/10 px-3 py-2 text-sm break-words text-red-200 ring-1 ring-red-400/40 ring-inset"
              >
                {testError}
              </p>
            </div>
          )}
        </Panel>
      )}

      {confirmingClear && (
        <Modal
          title="Forget this server?"
          onClose={() => {
            setConfirmingClear(false);
          }}
        >
          <p className="text-sm text-slate-300">
            No more codes can be sent. New players wait until a server is set up again, or until you
            let them in.
          </p>
          <div className="mt-6 flex gap-3">
            <Button variant="danger" disabled={pending} onClick={clear}>
              Forget it
            </Button>
            <Button
              variant="quiet"
              onClick={() => {
                setConfirmingClear(false);
              }}
            >
              Keep it
            </Button>
          </div>
        </Modal>
      )}
    </div>
  );
}
