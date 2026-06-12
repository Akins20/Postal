import type { HTMLAttributes } from "react";

import { cn } from "@/lib/cn";

/**
 * A vibrancy "window" panel - the macOS frosted-glass surface for cards/sheets
 * (FRONTEND_PLAN §5/§5.1). Falls back to an opaque surface where backdrop-filter
 * or transparency is unavailable (handled in globals.css).
 */
export function Panel({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("material-panel shadow-window rounded-xl", className)} {...props} />;
}
