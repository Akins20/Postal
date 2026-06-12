import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { cn } from "@/lib/cn";

import { Icon } from "./icon";

/**
 * A guided empty state (FRONTEND_PLAN §11) - explains what a surface is for and
 * the next action, so users aren't faced with a blank screen.
 */
export function EmptyState({
  icon,
  title,
  description,
  action,
  className,
}: {
  icon?: LucideIcon;
  title: string;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-3 px-6 py-16 text-center",
        className,
      )}
    >
      {icon && (
        <div className="bg-fg/5 text-fg-muted flex h-12 w-12 items-center justify-center rounded-full">
          <Icon icon={icon} size={24} />
        </div>
      )}
      <h2 className="text-fg text-base font-semibold">{title}</h2>
      {description && <p className="text-fg-muted max-w-sm text-sm">{description}</p>}
      {action && <div className="pt-1">{action}</div>}
    </div>
  );
}
