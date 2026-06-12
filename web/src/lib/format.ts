/**
 * A handle with exactly one leading "@". The backend already stores X handles
 * as "@name"; other platforms may not - never blindly prepend.
 */
export function atHandle(handle: string): string {
  return handle.startsWith("@") ? handle : `@${handle}`;
}

/** Human-readable byte size (binary units, one decimal above KiB). */
export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KiB", "MiB", "GiB"];
  let value = bytes;
  let unit = "B";
  for (const next of units) {
    if (value < 1024) break;
    value /= 1024;
    unit = next;
  }
  return `${value.toFixed(1)} ${unit}`;
}
