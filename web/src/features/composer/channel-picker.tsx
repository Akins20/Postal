"use client";

import { platformInfo } from "@/config/platforms";
import type { Channel } from "@/data/channels";
import { cn } from "@/lib/cn";
import { atHandle } from "@/lib/format";
import { Tooltip } from "@/ui/primitives/tooltip";

/**
 * Pick which connected channels a post goes to. Non-active channels are shown
 * but disabled (they can't accept publishes until reconnected).
 */
export function ChannelPicker({
  channels,
  selected,
  onToggle,
}: {
  channels: Channel[];
  selected: string[];
  onToggle: (channelId: string) => void;
}) {
  return (
    <fieldset className="flex flex-wrap gap-2">
      <legend className="text-fg mb-2 text-sm font-medium">Publish to</legend>
      {channels.map((c) => {
        const info = platformInfo(c.platform);
        const PlatformIcon = info.icon;
        const checked = selected.includes(c.id);
        const disabled = c.status !== "active";
        const chip = (
          <label
            key={c.id}
            className={cn(
              "border-separator focus-within:ring-ring flex cursor-pointer items-center gap-2 rounded-full border px-3 py-1.5 text-sm transition-colors focus-within:ring-2",
              checked ? "bg-accent/15 border-accent/40 text-fg" : "text-fg-muted hover:bg-fg/5",
              disabled && "cursor-not-allowed opacity-50",
            )}
          >
            <input
              type="checkbox"
              className="sr-only"
              checked={checked}
              disabled={disabled}
              onChange={() => onToggle(c.id)}
            />
            <PlatformIcon width={14} height={14} aria-hidden />
            <span>{atHandle(c.handle)}</span>
          </label>
        );
        return disabled ? (
          <Tooltip key={c.id} content="Reconnect this account on the Channels page to publish.">
            {chip}
          </Tooltip>
        ) : (
          chip
        );
      })}
    </fieldset>
  );
}
