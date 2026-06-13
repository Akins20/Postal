"use client";

import { Radio } from "lucide-react";

import { PLATFORMS } from "@/config/platforms";
import { useChannels } from "@/data/channels";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { EmptyState } from "@/ui/primitives/empty-state";
import { Hint } from "@/ui/primitives/hint";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";
import { StatusPill } from "@/ui/primitives/status-pill";
import { Tooltip } from "@/ui/primitives/tooltip";

import { ChannelCard } from "./channel-card";
import { ConnectChannelButton } from "./connect-channel-button";

/** The Channels screen: connected accounts plus the connect-a-platform list. */
export function ChannelsPanel() {
  const { active } = useActiveWorkspace();
  const { data: channels, isPending, isError } = useChannels(active?.id);
  // Platforms that already have at least one connected account, so the connect
  // list can show "Add another" instead of a plain "Connect".
  const connectedPlatforms = new Set((channels ?? []).map((c) => c.platform));

  if (!active) {
    return (
      <div className="py-10 text-center">
        <Spinner label="Loading workspace" />
      </div>
    );
  }

  return (
    <div className="grid items-start gap-6 lg:grid-cols-2">
      <Panel className="p-6">
        <div className="flex items-center gap-1.5">
          <h2 className="text-fg text-sm font-semibold">Connected accounts</h2>
          <Hint label="About connected accounts">
            Each connected account becomes a channel you can publish and schedule posts to.
          </Hint>
        </div>
        <p className="text-fg-muted mt-1 mb-2 text-sm">
          Social accounts this workspace can publish to.
        </p>
        {isPending && (
          <div className="py-6 text-center">
            <Spinner label="Loading channels" />
          </div>
        )}
        {isError && (
          <p role="alert" className="text-danger text-sm">
            Couldn&apos;t load channels. Please try again.
          </p>
        )}
        {channels?.length === 0 && (
          <EmptyState
            icon={Radio}
            title="No accounts connected yet"
            description="Connect your first social account below to start composing and scheduling posts."
            className="py-10"
          />
        )}
        {channels?.map((c) => (
          <ChannelCard key={c.id} workspaceId={active.id} channel={c} />
        ))}
      </Panel>

      <Panel className="p-6">
        <h2 className="text-fg text-sm font-semibold">Connect a platform</h2>
        <p className="text-fg-muted mt-1 mb-2 text-sm">
          You&apos;ll be sent to the platform to authorize Postal, then brought right back.
        </p>
        {PLATFORMS.map((p) => {
          const PlatformIcon = p.icon;
          const isConnected = connectedPlatforms.has(p.key);
          return (
            <div
              key={p.key}
              className="border-separator flex flex-wrap items-center gap-3 border-b py-3.5 last:border-0"
            >
              <div className="bg-fg/5 text-fg flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
                <PlatformIcon width={18} height={18} aria-hidden />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-fg flex items-center gap-2 text-sm font-medium">
                  {p.label}
                  {isConnected && <StatusPill tone="success">Connected</StatusPill>}
                  {p.payPerUse && (
                    <Tooltip content="The platform's API bills per request, so publishing here spends wallet credits. Every other platform is free.">
                      <span
                        tabIndex={0}
                        className="focus-visible:ring-ring rounded-full focus-visible:ring-2"
                      >
                        <StatusPill tone="warning">Pay-per-use</StatusPill>
                      </span>
                    </Tooltip>
                  )}
                </p>
                <p className="text-fg-muted text-xs">{p.hint}</p>
                {p.caveat && <p className="text-warning mt-0.5 text-xs">{p.caveat}</p>}
              </div>
              <ConnectChannelButton
                workspaceId={active.id}
                platform={p.key}
                label={isConnected ? "Add another" : "Connect"}
                variant={isConnected ? "secondary" : "primary"}
              />
            </div>
          );
        })}
      </Panel>
    </div>
  );
}
