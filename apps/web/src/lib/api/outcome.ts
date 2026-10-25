import { errorMessage, UNREACHABLE, type ApiError } from './client';

/** How a request from the browser went: its answer, or what to tell the player. */
export type Outcome<T> =
  { readonly ok: true; readonly value: T } | { readonly ok: false; readonly message: string };

interface Answer<T> {
  readonly data?: T;
  readonly error?: ApiError;
  readonly response: Response;
}

/** Sends a request, turning a refusal or a lost connection into a message for the player. */
export async function attempt<T>(send: () => Promise<Answer<T>>): Promise<Outcome<T>> {
  try {
    const { data, error, response } = await send();
    if (response.ok && data !== undefined) return { ok: true, value: data };
    return { ok: false, message: errorMessage(error) };
  } catch {
    return { ok: false, message: UNREACHABLE };
  }
}

/** Sends a request for something that may not exist: its answer, or null when not found. */
export async function attemptFound<T>(send: () => Promise<Answer<T>>): Promise<Outcome<T | null>> {
  try {
    const { data, error, response } = await send();
    if (response.ok && data !== undefined) return { ok: true, value: data };
    if (response.status === 404) return { ok: true, value: null };
    return { ok: false, message: errorMessage(error) };
  } catch {
    return { ok: false, message: UNREACHABLE };
  }
}

/** Sends a request answered with no content. */
export async function attemptEmpty(send: () => Promise<Answer<unknown>>): Promise<Outcome<null>> {
  try {
    const { error, response } = await send();
    if (response.ok) return { ok: true, value: null };
    return { ok: false, message: errorMessage(error) };
  } catch {
    return { ok: false, message: UNREACHABLE };
  }
}
