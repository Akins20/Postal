"use client";

import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { cn } from "@/lib/cn";
import { Icon } from "@/ui/primitives/icon";

export interface SidebarItem {
  href: string;
  label: string;
  icon?: LucideIcon;
}

export interface SidebarSection {
  title?: string;
  items: SidebarItem[];
}

/**
 * macOS source-list sidebar - navigation for a feature/sub-route (FRONTEND_PLAN
 * §5). Translucent, sectioned, with a selection highlight. Used persistently on
 * desktop/tablet and inside a slide-over sheet on mobile (see FeatureShell).
 */
export function Sidebar({
  sections,
  header,
  className,
}: {
  sections: SidebarSection[];
  header?: ReactNode;
  className?: string;
}) {
  const pathname = usePathname();

  return (
    <nav
      aria-label="Section"
      className={cn(
        "material-sidebar border-separator flex h-full w-60 shrink-0 flex-col gap-4 border-r p-3",
        className,
      )}
    >
      {header}
      {sections.map((section, i) => (
        <div key={section.title ?? i} className="flex flex-col gap-0.5">
          {section.title && (
            <p className="text-fg-subtle px-2 pb-1 text-xs font-medium tracking-wide uppercase">
              {section.title}
            </p>
          )}
          {section.items.map((item) => {
            const active = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "text-fg hover:bg-fg/5 focus-visible:ring-ring flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm transition-colors focus-visible:ring-2 focus-visible:outline-none",
                  active && "bg-accent/15 font-medium",
                )}
              >
                {item.icon && (
                  <Icon
                    icon={item.icon}
                    size={18}
                    className={active ? "text-accent" : "text-fg-muted"}
                  />
                )}
                <span className="truncate">{item.label}</span>
              </Link>
            );
          })}
        </div>
      ))}
    </nav>
  );
}
