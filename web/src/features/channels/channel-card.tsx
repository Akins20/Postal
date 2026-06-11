"use client";

import { useState } from "react";

import { platformInfo } from "@/config/platforms";
import { useDisconnectChannel, type Channel, type ChannelStatus } from "@/data/channels";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { ConfirmDialog } from "@/ui/primitives/confirm-dialog";
import { StatusPill } from "@/ui/primitives/status-pill";
import { Tooltip } from "@/ui/primitives/tooltip";

const STATUS: Record<ChannelStatus, { label: string; tone: "success" | "warning" | "danger" }> = {
  active: { label: "Active", tone: "success" },
  expired: { label: "Expired", tone: "warning" },
  revoked: { label: "Revoked", tone: "danger" },
};

const STATUS_HINT: Record<ChannelStatus, string> = {
  active: "This account is connected and ready to publish.",
  expired: "Access expired — reconnect the account to keep publishing.",
  revoked: "Access was revoked on the platform — reconnect to use it again.",
};

/** A connected social account row: identity, health, and disconnect. */
export function ChannelCard({ workspaceId, channel }: { workspaceId: string; channel: Channel }) {
  const info = platformInfo(channel.platform);
  const status = STATUS[channel.status];
  const disconnect = useDisconnectChannel(workspaceId);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const PlatformIcon = info.icon;

  const onDisconnect = async () => {
    setError(null);
    try {
      await disconnect.mutateAsync({ channelId: channel.id });
      setOpen(false);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <div className="border-separator flex flex-wrap items-center gap-3 border-b py-3.5 last:border-0">
      <div className="bg-fg/5 text-fg flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
        <PlatformIcon width={18} height={18} aria-hidden />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-fg truncate text-sm font-medium">{channel.display_name}</p>
        <p className="text-fg-muted truncate text-xs">
          @{channel.handle} · {info.label}
        </p>
      </div>
      <Tooltip content={STATUS_HINT[channel.status]}>
        <span tabIndex={0} className="focus-visible:ring-ring rounded-full focus-visible:ring-2">
          <StatusPill tone={status.tone}>{status.label}</StatusPill>
        </span>
      </Tooltip>
      <ConfirmDialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (!next) setError(null);
        }}
        trigger={
          <Button variant="secondary" size="sm">
            Disconnect
          </Button>
        }
        title={`Disconnect @${channel.handle}?`}
        description={
          <>
            Scheduled posts targeting this account will fail to publish until it is reconnected.
            {error && (
              <span role="alert" className="text-danger mt-2 block">
                {error}
              </span>
            )}
          </>
        }
        confirmLabel="Disconnect"
        destructive
        pending={disconnect.isPending}
        onConfirm={onDisconnect}
      />
    </div>
  );
}
