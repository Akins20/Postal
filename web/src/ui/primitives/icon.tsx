import type { LucideIcon } from "lucide-react";

import { cn } from "@/lib/cn";

/**
 * Icon wrapper: one family (lucide), consistent stroke + size defaults, and
 * sane a11y (decorative by default; pass `label` for a meaningful icon).
 */
export function Icon({
  icon: IconComponent,
  size = 20,
  label,
  className,
}: {
  icon: LucideIcon;
  size?: number;
  label?: string;
  className?: string;
}) {
  return (
    <IconComponent
      size={size}
      strokeWidth={1.75}
      aria-hidden={label ? undefined : true}
      aria-label={label}
      role={label ? "img" : undefined}
      className={cn("shrink-0", className)}
    />
  );
}
