import { forwardRef, useId, type InputHTMLAttributes, type ReactNode } from "react";

import { cn } from "@/lib/cn";

interface FormFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  hint?: ReactNode;
}

/**
 * An accessible labelled input (FRONTEND_PLAN §9.2): label tied via htmlFor,
 * errors announced (role="alert") and associated with aria-describedby +
 * aria-invalid, optional inline hint.
 */
export const FormField = forwardRef<HTMLInputElement, FormFieldProps>(function FormField(
  { label, error, hint, id, className, ...props },
  ref,
) {
  const autoId = useId();
  const fieldId = id ?? autoId;
  const errorId = `${fieldId}-error`;
  const hintId = `${fieldId}-hint`;
  const describedBy = cn(error && errorId, hint && hintId) || undefined;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={fieldId} className="text-fg text-sm font-medium">
        {label}
      </label>
      <input
        ref={ref}
        id={fieldId}
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy}
        className={cn(
          "border-separator bg-elevated text-fg placeholder:text-fg-subtle focus-visible:ring-ring h-10 rounded-md border px-3 text-sm transition-shadow outline-none focus-visible:ring-2",
          error && "border-danger",
          className,
        )}
        {...props}
      />
      {hint && !error && (
        <p id={hintId} className="text-fg-muted text-xs">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} role="alert" className="text-danger text-xs">
          {error}
        </p>
      )}
    </div>
  );
});
