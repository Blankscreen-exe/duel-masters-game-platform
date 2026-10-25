'use client';

import { useRouter } from 'next/navigation';
import { useState, type ReactNode, type SubmitEvent } from 'react';
import { Button } from '@/components/ui/button';
import { Modal } from '@/components/ui/modal';
import { SelectField } from '@/components/ui/select-field';
import { TextField } from '@/components/ui/text-field';
import type { AdminUser } from '@/lib/admin';
import {
  banUser,
  muteUser,
  setUserPassword,
  setUserRole,
  unbanUser,
  unmuteUser,
} from '@/lib/admin-actions';
import type { Outcome } from '@/lib/api/outcome';
import { PASSWORD_MAX_LENGTH, PASSWORD_MIN_LENGTH } from '@/lib/credentials';
import { textField } from '@/lib/form-data';
import {
  BAN_LENGTHS,
  BAN_REASON_MAX_LENGTH,
  MUTE_LENGTHS,
  penaltyEnd,
  type PenaltyLength,
} from '@/lib/moderation';
import { notify } from '@/lib/notifications';

type Step = 'menu' | 'ban' | 'unban' | 'mute' | 'unmute' | 'role' | 'password';

/**
 * The Manage button of a player in the admin panel's list, and its dialog: choose what to do, then
 * fill it in or confirm it. The list is drawn again once something is done.
 */
export function UserManager({ user }: { readonly user: AdminUser }) {
  const router = useRouter();
  const [step, setStep] = useState<Step | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function open(next: Step) {
    setError(null);
    setStep(next);
  }

  /** Does the chosen thing; on success says so, closes and draws the list again. */
  async function run<T>(send: () => Promise<Outcome<T>>, done: (value: T) => string) {
    setPending(true);
    setError(null);
    const outcome = await send();
    setPending(false);
    if (!outcome.ok) {
      setError(outcome.message);
      return;
    }
    notify(done(outcome.value));
    setStep(null);
    router.refresh();
  }

  const close = () => {
    setStep(null);
  };
  const name = user.username;
  const isAdmin = user.role === 'admin';

  return (
    <>
      <Button
        variant="secondary"
        className="px-3 py-1.5 text-xs"
        aria-label={`Manage ${name}`}
        onClick={() => {
          open('menu');
        }}
      >
        Manage
      </Button>

      {step !== null && (
        <Modal title={`Manage ${name}`} onClose={close}>
          {step === 'menu' && (
            <ul className="space-y-2">
              <MenuItem
                onClick={() => {
                  open(user.ban ? 'unban' : 'ban');
                }}
                disabled={!user.ban && isAdmin}
                hint={!user.ban && isAdmin ? 'Make them a player first.' : undefined}
              >
                {user.ban ? 'Lift the ban' : 'Ban…'}
              </MenuItem>
              <MenuItem
                onClick={() => {
                  open(user.mute ? 'unmute' : 'mute');
                }}
              >
                {user.mute ? 'Lift the mute' : 'Mute in the chat…'}
              </MenuItem>
              <MenuItem
                onClick={() => {
                  open('role');
                }}
                disabled={!isAdmin && user.ban !== undefined}
                hint={!isAdmin && user.ban ? 'Lift the ban first.' : undefined}
              >
                {isAdmin ? 'Make a player' : 'Make an admin'}
              </MenuItem>
              <MenuItem
                onClick={() => {
                  open('password');
                }}
              >
                Give a new password…
              </MenuItem>
            </ul>
          )}

          {step === 'ban' && (
            <PenaltyForm
              lengths={BAN_LENGTHS}
              defaultLength="7 days"
              withReason
              submitLabel="Ban"
              pending={pending}
              onCancel={close}
              onSubmit={(length, reason) =>
                void run(
                  () => banUser(user.id, reason, length.minutes),
                  (banned) => `${name} is banned ${penaltyEnd(banned.ban?.until)}.`,
                )
              }
            >
              {name} is logged out at once, and their game stops without counting. They are shown
              the reason when they try to log in.
            </PenaltyForm>
          )}

          {step === 'mute' && (
            <PenaltyForm
              lengths={MUTE_LENGTHS}
              defaultLength="24 hours"
              submitLabel="Mute"
              pending={pending}
              onCancel={close}
              onSubmit={(length) =>
                void run(
                  () => muteUser(user.id, length.minutes),
                  (muted) => `${name} is muted ${penaltyEnd(muted.mute?.until)}.`,
                )
              }
            >
              {name} can still read the chat and play, but cannot post.
            </PenaltyForm>
          )}

          {step === 'unban' && (
            <Confirm
              label="Lift the ban"
              pending={pending}
              onCancel={close}
              onConfirm={() =>
                void run(
                  () => unbanUser(user.id),
                  () => `${name}'s ban is lifted.`,
                )
              }
            >
              {name} can log in and play again at once.
            </Confirm>
          )}

          {step === 'unmute' && (
            <Confirm
              label="Lift the mute"
              pending={pending}
              onCancel={close}
              onConfirm={() =>
                void run(
                  () => unmuteUser(user.id),
                  () => `${name}'s mute is lifted.`,
                )
              }
            >
              {name} can post in the chat again at once.
            </Confirm>
          )}

          {step === 'role' && (
            <Confirm
              label={isAdmin ? 'Make a player' : 'Make an admin'}
              pending={pending}
              onCancel={close}
              onConfirm={() =>
                void run(
                  () => setUserRole(user.id, isAdmin ? 'player' : 'admin'),
                  (changed) =>
                    changed.role === 'admin'
                      ? `${name} is now an admin.`
                      : `${name} is now a player.`,
                )
              }
            >
              {isAdmin
                ? `${name} will no longer see the admin panel.`
                : `${name} will be able to do everything in the admin panel, including to you.`}
            </Confirm>
          )}

          {step === 'password' && (
            <PasswordForm
              name={name}
              pending={pending}
              onCancel={close}
              onSubmit={(password) =>
                void run(
                  () => setUserPassword(user.id, password),
                  () => `${name} has a new password. Pass it on to them.`,
                )
              }
            />
          )}

          {error && (
            <p
              role="alert"
              className="mt-4 bg-red-500/10 px-3 py-2 text-sm text-red-200 ring-1 ring-red-400/40 ring-inset"
            >
              {error}
            </p>
          )}
        </Modal>
      )}
    </>
  );
}

function MenuItem({
  onClick,
  disabled = false,
  hint,
  children,
}: {
  readonly onClick: () => void;
  readonly disabled?: boolean;
  readonly hint?: string;
  readonly children: ReactNode;
}) {
  return (
    <li>
      <Button
        variant="secondary"
        className="w-full justify-start"
        disabled={disabled}
        onClick={onClick}
      >
        {children}
      </Button>
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </li>
  );
}

interface StepProps {
  readonly pending: boolean;
  readonly onCancel: () => void;
}

/**
 * The dialog's own buttons: Cancel and the action. Cancel is a plain button, as it may sit inside
 * the step's form.
 */
function Actions({
  label,
  pending,
  danger = false,
  onClick,
  onCancel,
}: StepProps & {
  readonly label: string;
  readonly danger?: boolean;
  /** Without it, the action submits its form. */
  readonly onClick?: () => void;
}) {
  return (
    <div className="mt-6 flex justify-end gap-2">
      <Button variant="secondary" disabled={pending} onClick={onCancel}>
        Cancel
      </Button>
      <Button
        type={onClick ? 'button' : 'submit'}
        variant={danger ? 'danger' : 'primary'}
        disabled={pending}
        onClick={onClick}
      >
        {pending ? 'Working…' : label}
      </Button>
    </div>
  );
}

function Confirm({
  label,
  pending,
  onCancel,
  onConfirm,
  children,
}: StepProps & {
  readonly label: string;
  readonly onConfirm: () => void;
  readonly children: ReactNode;
}) {
  return (
    <>
      <p className="text-sm text-slate-300">{children}</p>
      <Actions label={label} pending={pending} onCancel={onCancel} onClick={onConfirm} />
    </>
  );
}

function PenaltyForm({
  lengths,
  defaultLength,
  withReason = false,
  submitLabel,
  pending,
  onCancel,
  onSubmit,
  children,
}: StepProps & {
  readonly lengths: readonly PenaltyLength[];
  readonly defaultLength: string;
  readonly withReason?: boolean;
  readonly submitLabel: string;
  readonly onSubmit: (length: PenaltyLength, reason: string) => void;
  readonly children: ReactNode;
}) {
  function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const length = lengths.find(({ label }) => label === textField(form, 'length'));
    if (length) onSubmit(length, textField(form, 'reason'));
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <p className="text-sm text-slate-300">{children}</p>
      <SelectField label="How long" name="length" defaultValue={defaultLength}>
        {lengths.map(({ label }) => (
          <option key={label} value={label}>
            {label}
          </option>
        ))}
      </SelectField>
      {withReason && (
        <TextField
          label="Reason"
          name="reason"
          required
          maxLength={BAN_REASON_MAX_LENGTH}
          hint="The player is shown this."
        />
      )}
      <Actions label={submitLabel} pending={pending} onCancel={onCancel} danger />
    </form>
  );
}

function PasswordForm({
  name,
  pending,
  onCancel,
  onSubmit,
}: StepProps & {
  readonly name: string;
  readonly onSubmit: (password: string) => void;
}) {
  const [mismatch, setMismatch] = useState(false);

  function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    const password = textField(form, 'password');
    const repeated = textField(form, 'repeat');
    setMismatch(password !== repeated);
    if (password === repeated) onSubmit(password);
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <p className="text-sm text-slate-300">
        For a player who has forgotten theirs. Pass the new password on to {name}; they are logged
        out everywhere, and can change it once they log in.
      </p>
      <TextField
        label="New password"
        name="password"
        type="password"
        autoComplete="new-password"
        required
        minLength={PASSWORD_MIN_LENGTH}
        maxLength={PASSWORD_MAX_LENGTH}
        hint={`At least ${String(PASSWORD_MIN_LENGTH)} characters`}
      />
      <TextField
        label="Repeat it"
        name="repeat"
        type="password"
        autoComplete="new-password"
        required
        maxLength={PASSWORD_MAX_LENGTH}
      />
      {mismatch && (
        <p role="alert" className="text-sm text-red-300">
          The two passwords are not the same.
        </p>
      )}
      <Actions label="Set the password" pending={pending} onCancel={onCancel} />
    </form>
  );
}
