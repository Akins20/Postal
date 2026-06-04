"use client";

import { Moon, Sun } from "lucide-react";
import { useTheme } from "next-themes";

import { cn } from "@/lib/cn";

import { Icon } from "./primitives/icon";
import { Tooltip } from "./primitives/tooltip";

/**
 * Light/dark toggle (FRONTEND_PLAN §5). Icon visibility is CSS-driven by the
 * `.dark` class next-themes sets before paint, so there's no hydration flash and
 * no mount-flag effect needed.
 */
export function ThemeToggle({ className }: { className?: string }) {
  const { resolvedTheme, setTheme } = useTheme();

  return (
    <Tooltip content="Toggle light / dark">
      <button
        type="button"
        aria-label="Toggle color theme"
        onClick={() => setTheme(resolvedTheme === "dark" ? "light" : "dark")}
        className={cn(
          "text-fg hover:bg-fg/8 focus-visible:ring-ring inline-flex h-9 w-9 items-center justify-center rounded-full transition-colors focus-visible:ring-2 focus-visible:outline-none",
          className,
        )}
      >
        <Icon icon={Moon} size={18} className="dark:hidden" />
        <Icon icon={Sun} size={18} className="hidden dark:block" />
      </button>
    </Tooltip>
  );
}
