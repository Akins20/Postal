"use client";

import Link from "next/link";

import { UserMenu } from "@/features/auth/user-menu";
import { WorkspaceSwitcher } from "@/features/workspace/workspace-switcher";
import { ThemeToggle } from "@/ui/theme-toggle";

/**
 * The global app chrome shown on every authenticated page: brand (a way back
 * to the dashboard from anywhere), workspace switcher, theme toggle, account
 * menu. The bottom dock handles destination switching; this bar handles
 * identity and context.
 */
export function AppHeader() {
  return (
    <header className="material-sidebar border-separator z-30 flex h-14 shrink-0 items-center justify-between gap-2 border-b px-4">
      <div className="flex min-w-0 items-center gap-2">
        <Link
          href="/"
          className="text-fg focus-visible:ring-ring rounded-md text-base font-semibold tracking-tight focus-visible:ring-2 focus-visible:outline-none"
        >
          Postal
        </Link>
        <span aria-hidden className="text-fg-subtle">
          /
        </span>
        <WorkspaceSwitcher />
      </div>
      <div className="flex items-center gap-1">
        <ThemeToggle />
        <UserMenu />
      </div>
    </header>
  );
}
