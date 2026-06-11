"use client";

import { useState } from "react";

import { useConnectChannel } from "@/data/channels";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";

/**
 * Starts the OAuth connect flow for a platform: asks the backend for the
 * authorization URL, then sends the browser there. The IdP returns the user
 * to /oauth/callback.
 */
export function ConnectChannelButton({
  workspaceId,
  platform,
  label = "Connect",
}: {
  workspaceId: string;
  platform: string;
  label?: string;
}) {
  const connect = useConnectChannel(workspaceId);
  const [error, setError] = useState<string | null>(null);

  const onClick = async () => {
    setError(null);
    try {
      const url = await connect.mutateAsync({ platform });
      window.location.assign(url);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <div className="flex flex-col items-end gap-1.5">
      <Button onClick={onClick} disabled={connect.isPending}>
        {connect.isPending ? "Redirecting…" : label}
      </Button>
      {error && (
        <p role="alert" className="text-danger text-xs">
          {error}
        </p>
      )}
    </div>
  );
}
