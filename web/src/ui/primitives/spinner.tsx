import { cn } from "@/lib/cn";

/** An accessible loading spinner (announces "Loading" to screen readers). */
export function Spinner({ className, label = "Loading" }: { className?: string; label?: string }) {
  return (
    <span
      role="status"
      aria-label={label}
      className={cn(
        "border-fg/20 border-t-fg inline-block h-5 w-5 animate-spin rounded-full border-2",
        className,
      )}
    />
  );
}
