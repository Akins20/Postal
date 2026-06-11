"use client";

import { subDays } from "date-fns";
import { useMemo, useState } from "react";

import { useMetricSeries, usePostAnalytics } from "@/data/analytics";
import type { Channel } from "@/data/channels";
import { Spinner } from "@/ui/primitives/spinner";

import { SeriesChart } from "./series-chart";

const RANGES = [
  { days: 7, label: "7d" },
  { days: 30, label: "30d" },
  { days: 90, label: "90d" },
];

/**
 * One post's metrics: per-channel latest numbers plus a metric time series
 * with channel/metric/range pickers.
 */
export function PostDetail({
  workspaceId,
  postId,
  channels,
}: {
  workspaceId: string;
  postId: string;
  channels: Channel[];
}) {
  const { data: perChannel, isPending } = usePostAnalytics(workspaceId, postId);
  const channelById = new Map(channels.map((c) => [c.id, c]));

  const [channelId, setChannelId] = useState<string | undefined>(undefined);
  const [metric, setMetric] = useState<string | undefined>(undefined);
  const [days, setDays] = useState(30);

  // Default selection: first channel and its first metric, once loaded.
  const activeChannelId = channelId ?? perChannel?.[0]?.channel_id;
  const activeRow = perChannel?.find((c) => c.channel_id === activeChannelId);
  const metricNames = Object.keys(activeRow?.metrics ?? {}).sort();
  const activeMetric = metric && metricNames.includes(metric) ? metric : metricNames[0];

  const range = useMemo(() => {
    const now = new Date();
    return { from: subDays(now, days).toISOString(), to: now.toISOString() };
  }, [days]);

  const series = useMetricSeries({
    workspaceId,
    postId,
    channelId: activeChannelId,
    metric: activeMetric ?? "",
    from: range.from,
    to: range.to,
  });

  if (isPending) {
    return (
      <div className="py-6 text-center">
        <Spinner label="Loading post metrics" />
      </div>
    );
  }
  if (!perChannel || perChannel.length === 0) {
    return <p className="text-fg-muted py-4 text-sm">No metrics captured for this post yet.</p>;
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="grid gap-3 sm:grid-cols-2">
        {perChannel.map((row) => {
          const channel = channelById.get(row.channel_id);
          return (
            <div key={row.channel_id} className="border-separator rounded-lg border p-4">
              <p className="text-fg text-sm font-medium">
                {channel ? `@${channel.handle}` : row.channel_id.slice(0, 8)}
              </p>
              <dl className="mt-2 flex flex-wrap gap-x-5 gap-y-1.5">
                {Object.entries(row.metrics)
                  .sort(([a], [b]) => a.localeCompare(b))
                  .map(([name, value]) => (
                    <div key={name}>
                      <dt className="text-fg-subtle text-xs capitalize">{name}</dt>
                      <dd className="text-fg text-lg font-semibold tabular-nums">{value}</dd>
                    </div>
                  ))}
              </dl>
            </div>
          );
        })}
      </div>

      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-xs">
          <span className="text-fg font-medium">Channel</span>
          <select
            value={activeChannelId ?? ""}
            onChange={(e) => {
              setChannelId(e.target.value);
              setMetric(undefined);
            }}
            className="border-separator bg-elevated text-fg focus-visible:ring-ring h-9 rounded-md border px-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
          >
            {perChannel.map((row) => (
              <option key={row.channel_id} value={row.channel_id}>
                {channelById.get(row.channel_id)?.handle ?? row.channel_id.slice(0, 8)}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-xs">
          <span className="text-fg font-medium">Metric</span>
          <select
            value={activeMetric ?? ""}
            onChange={(e) => setMetric(e.target.value)}
            className="border-separator bg-elevated text-fg focus-visible:ring-ring h-9 rounded-md border px-2 text-sm capitalize focus-visible:ring-2 focus-visible:outline-none"
          >
            {metricNames.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        </label>
        <div
          role="tablist"
          aria-label="Time range"
          className="bg-fg/5 ml-auto flex rounded-lg p-0.5"
        >
          {RANGES.map((r) => (
            <button
              key={r.days}
              role="tab"
              type="button"
              aria-selected={days === r.days}
              onClick={() => setDays(r.days)}
              className={
                days === r.days
                  ? "bg-elevated text-fg focus-visible:ring-ring rounded-md px-3 py-1 text-sm shadow-sm focus-visible:ring-2 focus-visible:outline-none"
                  : "text-fg-muted focus-visible:ring-ring rounded-md px-3 py-1 text-sm focus-visible:ring-2 focus-visible:outline-none"
              }
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>

      {series.isPending && (
        <div className="py-8 text-center">
          <Spinner label="Loading series" />
        </div>
      )}
      {series.data && series.data.length === 0 && (
        <p className="text-fg-muted py-4 text-sm">No data points in this range.</p>
      )}
      {series.data && series.data.length > 0 && activeMetric && (
        <SeriesChart points={series.data} metric={activeMetric} />
      )}
    </div>
  );
}
