"use client";

import { addDays, format, isSameDay } from "date-fns";
import { CalendarClock, Radio, SquarePen } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { useChannels } from "@/data/channels";
import { usePosts } from "@/data/posts";
import { useCalendar } from "@/data/schedule";
import { JOB_TONE } from "@/features/schedule/job-item";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { Button } from "@/ui/primitives/button";
import { Icon } from "@/ui/primitives/icon";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";
import { StatusPill } from "@/ui/primitives/status-pill";

function WidgetHeader({
  icon,
  title,
  href,
  linkLabel,
}: {
  icon: typeof Radio;
  title: string;
  href: string;
  linkLabel: string;
}) {
  return (
    <div className="mb-3 flex items-center justify-between gap-2">
      <h2 className="text-fg flex items-center gap-2 text-sm font-semibold">
        <Icon icon={icon} size={16} className="text-accent" />
        {title}
      </h2>
      <Button asChild variant="ghost" size="sm">
        <Link href={href}>{linkLabel}</Link>
      </Button>
    </div>
  );
}

/** Dashboard widgets: what's next, drafts in progress, channel health. */
export function OverviewWidgets() {
  const { active } = useActiveWorkspace();
  // Pinned per mount: a fresh Date each render would change the calendar
  // query key every render and the query would never settle.
  const [now] = useState(() => new Date());
  const { data: jobs, isPending: jobsPending } = useCalendar(
    active?.id,
    now.toISOString(),
    addDays(now, 7).toISOString(),
  );
  const { data: posts, isPending: postsPending } = usePosts(active?.id);
  const { data: channels, isPending: channelsPending } = useChannels(active?.id);

  if (!active) {
    return (
      <div className="py-10 text-center">
        <Spinner label="Loading workspace" />
      </div>
    );
  }

  const upcoming = (jobs ?? [])
    .filter((j) => j.status === "scheduled")
    .sort((a, b) => a.run_at.localeCompare(b.run_at))
    .slice(0, 5);
  const drafts = (posts ?? []).filter((p) => p.status === "draft").slice(0, 5);
  const channelByID = new Map((channels ?? []).map((c) => [c.id, c]));
  const needsAttention = (channels ?? []).filter((c) => c.status !== "active");

  return (
    <section aria-label="Overview" className="grid grid-cols-1 gap-3 lg:grid-cols-3">
      <Panel className="p-5">
        <WidgetHeader
          icon={CalendarClock}
          title="Next 7 days"
          href="/calendar"
          linkLabel="Calendar"
        />
        {jobsPending && <Spinner label="Loading schedule" className="mx-auto block" />}
        {!jobsPending && upcoming.length === 0 && (
          <p className="text-fg-muted text-sm">Nothing scheduled yet.</p>
        )}
        <ul className="flex list-none flex-col gap-2">
          {upcoming.map((j) => (
            <li key={j.id} className="flex items-center gap-2 text-sm">
              <span className="text-fg-muted w-24 shrink-0 text-xs tabular-nums">
                {isSameDay(new Date(j.run_at), now)
                  ? format(new Date(j.run_at), "'Today' HH:mm")
                  : format(new Date(j.run_at), "EEE d · HH:mm")}
              </span>
              <span className="text-fg min-w-0 flex-1 truncate">
                @{channelByID.get(j.channel_id)?.handle ?? j.channel_id.slice(0, 8)}
              </span>
              <StatusPill tone={JOB_TONE[j.status]}>{j.status}</StatusPill>
            </li>
          ))}
        </ul>
      </Panel>

      <Panel className="p-5">
        <WidgetHeader icon={SquarePen} title="Drafts" href="/compose" linkLabel="Compose" />
        {postsPending && <Spinner label="Loading drafts" className="mx-auto block" />}
        {!postsPending && drafts.length === 0 && (
          <p className="text-fg-muted text-sm">No drafts — write something.</p>
        )}
        <ul className="flex list-none flex-col gap-2">
          {drafts.map((p) => (
            <li key={p.id} className="text-fg truncate text-sm">
              {p.variants?.[0]?.body || <em className="text-fg-muted">(no text)</em>}
            </li>
          ))}
        </ul>
      </Panel>

      <Panel className="p-5">
        <WidgetHeader icon={Radio} title="Channels" href="/channels" linkLabel="Manage" />
        {channelsPending && <Spinner label="Loading channels" className="mx-auto block" />}
        {!channelsPending && (channels ?? []).length === 0 && (
          <p className="text-fg-muted text-sm">No accounts connected yet.</p>
        )}
        {channels && channels.length > 0 && (
          <div className="flex flex-col gap-2 text-sm">
            <p className="text-fg">
              {channels.length} connected
              {needsAttention.length > 0 && (
                <span className="text-warning"> · {needsAttention.length} need attention</span>
              )}
            </p>
            <ul className="flex list-none flex-col gap-1.5">
              {channels.slice(0, 4).map((c) => (
                <li key={c.id} className="flex items-center justify-between gap-2">
                  <span className="text-fg-muted truncate text-xs">@{c.handle}</span>
                  <StatusPill tone={c.status === "active" ? "success" : "warning"}>
                    {c.status}
                  </StatusPill>
                </li>
              ))}
            </ul>
          </div>
        )}
      </Panel>
    </section>
  );
}
