"use client";

import { useEffect } from "react";

import { logger } from "@/lib/logger";

import "./globals.css";

/**
 * Global error boundary - catches errors in the root layout itself, so it must
 * render its own <html>/<body> (FRONTEND_PLAN §8).
 */
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    logger.error("global error boundary", { digest: error.digest, message: error.message });
  }, [error]);

  return (
    <html lang="en" suppressHydrationWarning>
      <body className="min-h-screen antialiased">
        <div className="flex min-h-screen items-center justify-center p-6">
          <div
            role="alert"
            className="border-separator bg-elevated shadow-window flex max-w-md flex-col items-center gap-4 rounded-xl border p-8 text-center"
          >
            <h1 className="text-fg text-lg font-semibold">Something went wrong</h1>
            <p className="text-fg-muted text-sm">
              The app hit an unexpected error. Please try again.
            </p>
            <button
              type="button"
              onClick={reset}
              className="bg-accent text-accent-fg focus-visible:ring-ring inline-flex h-10 items-center justify-center rounded-md px-4 text-sm font-medium hover:brightness-110 focus-visible:ring-2 focus-visible:outline-none"
            >
              Try again
            </button>
          </div>
        </div>
      </body>
    </html>
  );
}
