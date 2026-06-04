"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { useVerifyEmail } from "@/data/auth";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";

type Status = "verifying" | "success" | "error";

export function VerifyEmailClient({ token }: { token: string }) {
  const verify = useVerifyEmail();
  const [status, setStatus] = useState<Status>(token ? "verifying" : "error");
  const [message, setMessage] = useState(token ? "" : "This link is missing its token.");
  const ran = useRef(false);

  useEffect(() => {
    if (!token || ran.current) return;
    ran.current = true;
    verify
      .mutateAsync({ token })
      .then(() => setStatus("success"))
      .catch((e: NormalizedError) => {
        setStatus("error");
        setMessage(e.message);
      });
  }, [token, verify]);

  return (
    <div role="status" aria-live="polite" className="flex flex-col items-center gap-4 text-center">
      {status === "verifying" && <p className="text-fg-muted text-sm">Verifying your email…</p>}
      {status === "success" && (
        <>
          <p className="text-fg-muted text-sm">Your email address is verified.</p>
          <Button asChild>
            <Link href="/login">Sign in</Link>
          </Button>
        </>
      )}
      {status === "error" && (
        <>
          <p className="text-danger text-sm">{message || "Verification failed."}</p>
          <Button asChild variant="secondary">
            <Link href="/login">Back to sign in</Link>
          </Button>
        </>
      )}
    </div>
  );
}
