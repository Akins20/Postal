"use client";

import { useEffect, useState } from "react";

import { useResendVerification } from "@/data/auth";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";

const COOLDOWN_SECONDS = 60;

/**
 * A "resend verification email" button with a cooldown timer. After a send (or a
 * rate-limit response) the button is disabled and counts down, so users cannot
 * spam the endpoint and get clear feedback that the mail is on its way.
 */
export function ResendVerification({ email }: { email: string }) {
  const resend = useResendVerification();
  const [secondsLeft, setSecondsLeft] = useState(0);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (secondsLeft <= 0) return;
    const timer = setInterval(() => setSecondsLeft((s) => s - 1), 1000);
    return () => clearInterval(timer);
  }, [secondsLeft]);

  const onClick = async () => {
    setError(null);
    try {
      await resend.mutateAsync({ email });
      setSent(true);
    } catch (e) {
      setError((e as NormalizedError).message);
    } finally {
      // Cool down on both success and rate-limit so the button cannot be spammed.
      setSecondsLeft(COOLDOWN_SECONDS);
    }
  };

  const disabled = secondsLeft > 0 || resend.isPending || !email;
  const label =
    secondsLeft > 0
      ? `Resend in ${secondsLeft}s`
      : resend.isPending
        ? "Sending…"
        : "Resend verification email";

  return (
    <div className="flex flex-col items-center gap-1.5">
      <Button variant="secondary" onClick={onClick} disabled={disabled} className="w-full">
        {label}
      </Button>
      {sent && !error && (
        <p className="text-fg-muted text-xs">Sent. Check your inbox, and your spam folder.</p>
      )}
      {error && <p className="text-danger text-xs">{error}</p>}
    </div>
  );
}
