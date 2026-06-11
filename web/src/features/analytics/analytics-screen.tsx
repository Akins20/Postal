"use client";

import { format } from "date-fns";
import { BarChart3, Download } from "lucide-react";
import { useState } from "react";

import { analyticsCsvURL, useAnalyticsOverview } from "@/data/analytics";
import { useChannels } from "@/data/channels";
import { usePosts } from "@/data/posts";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { cn } from "@/lib/cn";
import { Button } from "@/ui/primitives/button";
import { EmptyState } from "@/ui/primitives/empty-state";
import { Hint } from "@/ui/primitives/hint";
import { Icon } from "@/ui/primitives/icon";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";

import { PostDetail } from "./post-detail";

/** The Analytics screen: workspace overview table + drill-down per post. */
export function AnalyticsScreen() {
  const { active } = useActiveWorkspace();
  const { data: rows, isPending, isError } = useAnalyticsOverview(active?.id);
  const { data: channels } = useChannels(active?.id);
  const { data: posts } = usePosts(active?.id);
  const [selectedPostId, setSelectedPostId] = useState<string | null>(null);

  if (!active) {
    return (
      <div className="py-10 text-center">
        <Spinner label="Loading workspace" />
      </div>
    );
  }

  const channelById = new Map((channels ?? []).map((c) => [c.id, c]));
  const excerptByPost = new Map(
    (posts ?? []).map((p) => [p.id, p.variants?.[0]?.body ?? "(no text)"]),
  );

  return (
    <div className="flex flex-col gap-6">
      <Panel className="p-6">
        <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
          <div>
            <div className="flex items-center gap-1.5">
              <h2 className="text-fg text-sm font-semibold">Published posts</h2>
              <Hint label="About these numbers">
                Latest captured numbers per post and channel. Metrics refresh periodically after a
                post publishes — pick a row for per-channel breakdown and trends.
              </Hint>
            </div>
            <p className="text-fg-muted mt-1 text-sm">Latest metrics per post and channel.</p>
          </div>
          <Button asChild variant="secondary" size="sm">
            <a href={analyticsCsvURL(active.id)} download>
              <Icon icon={Download} size={15} />
              Export CSV
            </a>
          </Button>
        </div>

        {isPending && (
          <div className="py-6 text-center">
            <Spinner label="Loading analytics" />
          </div>
        )}
        {isError && (
          <p role="alert" className="text-danger text-sm">
            Couldn&apos;t load analytics. Please try again.
          </p>
        )}
        {rows?.length === 0 && (
          <EmptyState
            icon={BarChart3}
            title="No metrics yet"
            description="Numbers show up here once scheduled posts publish and their first metrics are captured."
            className="py-10"
          />
        )}
        {rows && rows.length > 0 && (
          <ul className="flex list-none flex-col">
            {rows.map((row) => {
              const channel = channelById.get(row.channel_id);
              const selected = selectedPostId === row.post_id;
              return (
                <li key={`${row.post_id}-${row.channel_id}`}>
                  <button
                    type="button"
                    onClick={() => setSelectedPostId(selected ? null : row.post_id)}
                    aria-expanded={selected}
                    className={cn(
                      "border-separator focus-visible:ring-ring flex w-full flex-wrap items-center gap-3 border-b px-1 py-3 text-left transition-colors focus-visible:ring-2 focus-visible:outline-none",
                      selected ? "bg-accent/8" : "hover:bg-fg/4",
                    )}
                  >
                    <span className="text-fg min-w-0 flex-1 truncate text-sm">
                      {excerptByPost.get(row.post_id) ?? row.post_id.slice(0, 8)}
                    </span>
                    <span className="text-fg-muted text-xs">
                      {channel ? `@${channel.handle}` : row.channel_id.slice(0, 8)}
                    </span>
                    <span className="text-fg-muted text-xs tabular-nums">
                      {Object.entries(row.metrics)
                        .sort(([a], [b]) => a.localeCompare(b))
                        .map(([k, v]) => `${k} ${v}`)
                        .join(" · ")}
                    </span>
                    <span className="text-fg-subtle text-xs">
                      {format(new Date(row.captured_at), "d MMM HH:mm")}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </Panel>

      {selectedPostId && (
        <Panel className="p-6">
          <h2 className="text-fg mb-4 text-sm font-semibold">Post breakdown</h2>
          <PostDetail workspaceId={active.id} postId={selectedPostId} channels={channels ?? []} />
        </Panel>
      )}
    </div>
  );
}
