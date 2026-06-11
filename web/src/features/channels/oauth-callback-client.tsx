"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { useCompleteOAuth } from "@/data/channels";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";

/**
 * Finishes a channel connection after the IdP redirects back with state+code.
 * The state is single-use, so the exchange runs exactly once.
 */
export function OAuthCallbackClient({ state, code }: { state?: string; code?: string }) {
  const router = useRouter();
  const complete = useCompleteOAuth();
  const fired = useRef(false);
  const missing = !state || !code;
  const [result, setResult] = useState<{ status: "success" | "error"; message: string } | null>(
    null,
  );

  useEffect(() => {
    if (missing || fired.current) return;
    fired.current = true;
    complete
      .mutateAsync({ state, code })
      .then((channel) => {
        setResult({ status: "success", message: `@${channel.handle} is connected.` });
        router.replace("/channels");
      })
      .catch((e: NormalizedError) => {
        setResult({ status: "error", message: e.message });
      });
  }, [missing, state, code, complete, router]);

  const status = missing ? "error" : (result?.status ?? "connecting");
  const message = missing
    ? "This link is missing its authorization details. Try connecting again."
    : (result?.message ?? "");

  return (
    <div className="flex min-h-dvh items-center justify-center p-4">
      <Panel className="w-full max-w-sm p-7">
        <div
          role="status"
          aria-live="polite"
          className="flex flex-col items-center gap-4 text-center"
        >
          {status === "connecting" && (
            <>
              <Spinner label="Connecting account" />
              <p className="text-fg-muted text-sm">Connecting your account…</p>
            </>
          )}
          {status === "success" && <p className="text-fg-muted text-sm">{message} Redirecting…</p>}
          {status === "error" && (
            <>
              <p className="text-danger text-sm">{message || "The connection failed."}</p>
              <Button asChild variant="secondary">
                <Link href="/channels">Back to channels</Link>
              </Button>
            </>
          )}
        </div>
      </Panel>
    </div>
  );
}
