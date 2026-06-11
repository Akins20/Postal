"use client";

import { useId } from "react";

import { cn } from "@/lib/cn";

/**
 * A post body editor with a live character counter against the platform's cap
 * (over-limit is announced, not color-only — the count text flips too). The
 * server re-validates on save/publish; this is a compose-time aid.
 */
export function VariantEditor({
  label,
  value,
  onChange,
  charLimit,
  placeholder,
  disabled = false,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  charLimit?: number;
  placeholder?: string;
  disabled?: boolean;
}) {
  const id = useId();
  const remaining = charLimit !== undefined ? charLimit - value.length : undefined;
  const over = remaining !== undefined && remaining < 0;

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-baseline justify-between">
        <label htmlFor={id} className="text-fg text-sm font-medium">
          {label}
        </label>
        {remaining !== undefined && (
          <span
            aria-live="polite"
            className={cn(
              "text-xs tabular-nums",
              over ? "text-danger font-medium" : "text-fg-subtle",
            )}
          >
            {over ? `${-remaining} over the ${charLimit} limit` : `${remaining} left`}
          </span>
        )}
      </div>
      <textarea
        id={id}
        rows={5}
        value={value}
        disabled={disabled}
        placeholder={placeholder}
        aria-invalid={over || undefined}
        onChange={(e) => onChange(e.target.value)}
        className={cn(
          "border-separator bg-elevated text-fg placeholder:text-fg-subtle focus-visible:ring-ring resize-y rounded-md border px-3 py-2 text-sm transition-shadow outline-none focus-visible:ring-2",
          over && "border-danger",
        )}
      />
    </div>
  );
}
