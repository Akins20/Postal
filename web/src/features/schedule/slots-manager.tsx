"use client";

import { Trash2 } from "lucide-react";
import { useState } from "react";

import type { Channel } from "@/data/channels";
import { useCreateSlot, useDeleteSlot, useSlots } from "@/data/schedule";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { Hint } from "@/ui/primitives/hint";
import { Icon } from "@/ui/primitives/icon";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";

const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

/**
 * Per-channel weekly posting slots (what "next open slots" scheduling fills).
 * Day/time are interpreted in the slot's own timezone.
 */
export function SlotsManager({
  workspaceId,
  channels,
}: {
  workspaceId: string;
  channels: Channel[];
}) {
  const [channelId, setChannelId] = useState<string>(channels[0]?.id ?? "");
  const { data: slots, isPending } = useSlots(workspaceId, channelId || undefined);
  const create = useCreateSlot(workspaceId);
  const remove = useDeleteSlot(workspaceId);
  const [day, setDay] = useState(1);
  const [time, setTime] = useState("09:00");
  const [timezone, setTimezone] = useState(() => Intl.DateTimeFormat().resolvedOptions().timeZone);
  const [error, setError] = useState<string | null>(null);

  const add = async () => {
    setError(null);
    try {
      await create.mutateAsync({
        channel_id: channelId,
        day_of_week: day,
        time_of_day: time,
        timezone,
      });
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <Panel className="p-6">
      <div className="flex items-center gap-1.5">
        <h2 className="text-fg text-sm font-semibold">Posting slots</h2>
        <Hint label="About posting slots">
          A slot is a weekly time a channel likes to post (e.g. Monday 09:00). &quot;Next open
          slots&quot; scheduling fills the earliest free one per channel.
        </Hint>
      </div>
      <p className="text-fg-muted mt-1 mb-4 text-sm">
        The weekly schedule used by &quot;next open slots&quot;.
      </p>

      <label className="mb-3 flex flex-col gap-1 text-xs sm:max-w-60">
        <span className="text-fg font-medium">Channel</span>
        <select
          value={channelId}
          onChange={(e) => setChannelId(e.target.value)}
          className="border-separator bg-elevated text-fg focus-visible:ring-ring h-9 rounded-md border px-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
        >
          {channels.map((c) => (
            <option key={c.id} value={c.id}>
              @{c.handle} ({c.platform})
            </option>
          ))}
        </select>
      </label>

      {isPending && (
        <div className="py-4 text-center">
          <Spinner label="Loading slots" />
        </div>
      )}
      {slots?.length === 0 && (
        <p className="text-fg-muted py-2 text-sm">No slots yet for this channel.</p>
      )}
      {slots?.map((s) => (
        <div
          key={s.id}
          className="border-separator flex items-center justify-between gap-3 border-b py-2 text-sm last:border-0"
        >
          <span className="text-fg">
            {DAYS[s.day_of_week]} at {s.time_of_day}
            <span className="text-fg-subtle ml-2 text-xs">{s.timezone}</span>
          </span>
          <Button
            variant="ghost"
            size="sm"
            aria-label={`Delete ${DAYS[s.day_of_week]} ${s.time_of_day} slot`}
            disabled={remove.isPending}
            onClick={() => remove.mutate({ slotId: s.id, channelId: s.channel_id })}
          >
            <Icon icon={Trash2} size={15} />
          </Button>
        </div>
      ))}

      <div className="border-separator mt-4 flex flex-wrap items-end gap-3 border-t pt-4">
        <label className="flex flex-col gap-1 text-xs">
          <span className="text-fg font-medium">Day</span>
          <select
            value={day}
            onChange={(e) => setDay(Number(e.target.value))}
            className="border-separator bg-elevated text-fg focus-visible:ring-ring h-9 rounded-md border px-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
          >
            {DAYS.map((d, i) => (
              <option key={d} value={i}>
                {d}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-xs">
          <span className="text-fg font-medium">Time</span>
          <input
            type="time"
            value={time}
            onChange={(e) => setTime(e.target.value)}
            className="border-separator bg-elevated text-fg focus-visible:ring-ring h-9 rounded-md border px-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
          />
        </label>
        <label className="flex flex-col gap-1 text-xs">
          <span className="text-fg font-medium">Timezone</span>
          <select
            value={timezone}
            onChange={(e) => setTimezone(e.target.value)}
            className="border-separator bg-elevated text-fg focus-visible:ring-ring h-9 max-w-52 rounded-md border px-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
          >
            {Intl.supportedValuesOf("timeZone").map((tz) => (
              <option key={tz} value={tz}>
                {tz}
              </option>
            ))}
          </select>
        </label>
        <Button onClick={add} disabled={create.isPending || !channelId || !time}>
          {create.isPending ? "Adding…" : "Add slot"}
        </Button>
      </div>
      {error && (
        <p role="alert" className="text-danger mt-2 text-xs">
          {error}
        </p>
      )}
    </Panel>
  );
}
