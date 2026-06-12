"use client";

import { useEffect } from "react";

import { logger } from "@/lib/logger";
import { Button } from "@/ui/primitives/button";
import { Panel } from "@/ui/primitives/panel";

/**
 * Route-segment error boundary (FRONTEND_PLAN §8). Logs a structured error and
 * shows a recoverable, user-safe fallback.
 */
export default function RouteError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    logger.error("route error boundary", { digest: error.digest, message: error.message });
  }, [error]);

  return (
    <div className="flex min-h-[60vh] items-center justify-center p-6">
      <Panel role="alert" className="flex max-w-md flex-col items-center gap-4 p-8 text-center">
        <h1 className="text-fg text-lg font-semibold">Something went wrong</h1>
        <p className="text-fg-muted text-sm">
          An unexpected error occurred. You can try again - if it keeps happening, please reload the
          page.
        </p>
        <Button onClick={reset}>Try again</Button>
      </Panel>
    </div>
  );
}
