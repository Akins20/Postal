import type { FieldValues, Path, UseFormSetError } from "react-hook-form";

import type { NormalizedError } from "@/lib/api-error";

/**
 * Apply a normalized backend error to a react-hook-form: field errors map to the
 * matching inputs; returns true when at least one field error was applied (so the
 * caller can fall back to a form-level message otherwise). FRONTEND_PLAN §11.
 */
export function applyServerErrors<T extends FieldValues>(
  err: NormalizedError,
  setError: UseFormSetError<T>,
): boolean {
  const entries = Object.entries(err.fieldErrors);
  for (const [field, message] of entries) {
    setError(field as Path<T>, { message });
  }
  return entries.length > 0;
}
