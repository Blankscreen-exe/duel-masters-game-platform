import { useCallback, useState } from "react";

/** A yes or no preference kept in the browser between duels. Where storage is
 * unavailable, such as in some private browsing modes, the preference lasts
 * until the page is closed. */
export function useStoredFlag(
  key: string,
  defaultValue: boolean,
): [boolean, (value: boolean) => void] {
  const [value, setValue] = useState(() => readFlag(key, defaultValue));

  const update = useCallback(
    (next: boolean) => {
      setValue(next);

      try {
        window.localStorage.setItem(key, String(next));
      } catch {
        // Kept in memory for this page only.
      }
    },
    [key],
  );

  return [value, update];
}

function readFlag(key: string, defaultValue: boolean): boolean {
  try {
    const stored = window.localStorage.getItem(key);
    return stored === null ? defaultValue : stored === "true";
  } catch {
    return defaultValue;
  }
}
