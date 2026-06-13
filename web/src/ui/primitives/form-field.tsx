import { Eye, EyeOff } from "lucide-react";
import { forwardRef, useId, useState, type InputHTMLAttributes, type ReactNode } from "react";

import { cn } from "@/lib/cn";

interface FormFieldProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
  hint?: ReactNode;
}

/**
 * An accessible labelled input (FRONTEND_PLAN §9.2): label tied via htmlFor,
 * errors announced (role="alert") and associated with aria-describedby +
 * aria-invalid, optional inline hint. Password fields get a show/hide toggle.
 */
export const FormField = forwardRef<HTMLInputElement, FormFieldProps>(function FormField(
  { label, error, hint, id, className, type, ...props },
  ref,
) {
  const autoId = useId();
  const fieldId = id ?? autoId;
  const errorId = `${fieldId}-error`;
  const hintId = `${fieldId}-hint`;
  const describedBy = cn(error && errorId, hint && hintId) || undefined;

  // Password fields render a reveal toggle; the input type flips with it.
  const isPassword = type === "password";
  const [revealed, setRevealed] = useState(false);
  const inputType = isPassword ? (revealed ? "text" : "password") : type;

  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={fieldId} className="text-fg text-sm font-medium">
        {label}
      </label>
      <div className="relative">
        <input
          ref={ref}
          id={fieldId}
          type={inputType}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          className={cn(
            "border-separator bg-elevated text-fg placeholder:text-fg-subtle focus-visible:ring-ring h-10 w-full rounded-md border px-3 text-sm transition-shadow outline-none focus-visible:ring-2",
            isPassword && "pr-10",
            error && "border-danger",
            className,
          )}
          {...props}
        />
        {isPassword && (
          <button
            type="button"
            onClick={() => setRevealed((v) => !v)}
            aria-label={revealed ? "Hide password" : "Show password"}
            aria-pressed={revealed}
            tabIndex={-1}
            className="text-fg-subtle hover:text-fg focus-visible:ring-ring absolute inset-y-0 right-0 flex w-10 items-center justify-center rounded-r-md outline-none focus-visible:ring-2"
          >
            {revealed ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
          </button>
        )}
      </div>
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
