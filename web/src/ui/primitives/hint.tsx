"use client";

import { Info } from "lucide-react";
import type { ReactNode } from "react";

import { Icon } from "./icon";
import { Tooltip } from "./tooltip";

/**
 * An inline help affordance - a small info button that reveals a description on
 * hover/focus. Teaches in place without cluttering the UI (FRONTEND_PLAN §11).
 */
export function Hint({
  children,
  label = "More information",
}: {
  children: ReactNode;
  label?: string;
}) {
  return (
    <Tooltip content={children}>
      <button
        type="button"
        aria-label={label}
        className="text-fg-subtle hover:text-fg focus-visible:ring-ring inline-flex items-center justify-center rounded-full transition-colors focus-visible:ring-2 focus-visible:outline-none"
      >
        <Icon icon={Info} size={15} />
      </button>
    </Tooltip>
  );
}
