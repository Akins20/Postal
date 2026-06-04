"use client";

import * as Menu from "@radix-ui/react-dropdown-menu";
import { Check, ChevronsUpDown } from "lucide-react";

import { cn } from "@/lib/cn";
import { Icon } from "@/ui/primitives/icon";

import { useActiveWorkspace } from "./use-active-workspace";

/** Switch the active workspace (FRONTEND_PLAN §12.2). */
export function WorkspaceSwitcher({ className }: { className?: string }) {
  const { workspaces, active, setActive } = useActiveWorkspace();
  if (!active) return null;

  return (
    <Menu.Root>
      <Menu.Trigger asChild>
        <button
          type="button"
          aria-label="Switch workspace"
          className={cn(
            "text-fg hover:bg-fg/8 focus-visible:ring-ring flex items-center gap-1.5 rounded-md px-2 py-1 text-sm font-medium focus-visible:ring-2 focus-visible:outline-none",
            className,
          )}
        >
          <span className="max-w-[160px] truncate">{active.name}</span>
          <Icon icon={ChevronsUpDown} size={14} className="text-fg-muted" />
        </button>
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Content
          align="start"
          sideOffset={6}
          className="material-panel shadow-popover z-50 min-w-56 rounded-lg p-1"
        >
          <p className="text-fg-subtle px-2 py-1.5 text-xs font-medium tracking-wide uppercase">
            Workspaces
          </p>
          {workspaces.map((w) => (
            <Menu.Item
              key={w.id}
              onSelect={() => setActive(w.id)}
              className="text-fg data-[highlighted]:bg-fg/8 flex items-center justify-between gap-2 rounded-md px-2 py-1.5 text-sm outline-none"
            >
              <span className="truncate">{w.name}</span>
              {w.id === active.id && <Icon icon={Check} size={15} className="text-accent" />}
            </Menu.Item>
          ))}
        </Menu.Content>
      </Menu.Portal>
    </Menu.Root>
  );
}
