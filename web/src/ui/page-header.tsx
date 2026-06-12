import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";

import { Icon } from "@/ui/primitives/icon";

/**
 * Standard feature-page header: an accent icon chip, a tight title, a
 * one-line subtitle, and an optional action slot. Keeps hierarchy and
 * spacing identical across every screen.
 */
export function PageHeader({
  icon,
  title,
  subtitle,
  actions,
}: {
  icon: LucideIcon;
  title: string;
  subtitle?: string;
  actions?: ReactNode;
}) {
  return (
    <header className="flex flex-wrap items-center gap-3">
      <div className="from-accent-soft to-accent text-accent-fg flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-gradient-to-b shadow-[inset_0_1px_0_rgb(255_255_255/0.3),0_2px_6px_rgb(0_0_0/0.18)]">
        <Icon icon={icon} size={22} />
      </div>
      <div className="min-w-0 flex-1">
        <h1 className="text-fg text-xl font-semibold tracking-tight">{title}</h1>
        {subtitle && <p className="text-fg-muted mt-0.5 truncate text-sm">{subtitle}</p>}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </header>
  );
}
