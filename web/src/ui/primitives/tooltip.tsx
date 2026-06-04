"use client";

import * as RadixTooltip from "@radix-ui/react-tooltip";
import type { ReactNode } from "react";

import { cn } from "@/lib/cn";

/**
 * Contextual-help tooltip (FRONTEND_PLAN §11): keyboard- and screen-reader-
 * reachable via the Radix trigger. The provider lives in `app/providers`.
 */
export function Tooltip({
  content,
  children,
  side = "top",
  className,
}: {
  content: ReactNode;
  children: ReactNode;
  side?: "top" | "right" | "bottom" | "left";
  className?: string;
}) {
  return (
    <RadixTooltip.Root>
      <RadixTooltip.Trigger asChild>{children}</RadixTooltip.Trigger>
      <RadixTooltip.Portal>
        <RadixTooltip.Content
          side={side}
          sideOffset={8}
          collisionPadding={8}
          className={cn(
            "material-panel text-fg shadow-popover z-50 max-w-xs rounded-md px-2.5 py-1.5 text-xs leading-snug",
            className,
          )}
        >
          {content}
          <RadixTooltip.Arrow className="fill-elevated" width={11} height={5} />
        </RadixTooltip.Content>
      </RadixTooltip.Portal>
    </RadixTooltip.Root>
  );
}
