"use client";

import { format } from "date-fns";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from "recharts";

import type { SeriesPoint } from "@/data/analytics";

/**
 * Time-series area chart for one metric. Colors come from the design tokens
 * via CSS variables so light/dark themes both read correctly.
 */
export function SeriesChart({ points, metric }: { points: SeriesPoint[]; metric: string }) {
  const data = points.map((p) => ({
    time: format(new Date(p.captured_at), "d MMM HH:mm"),
    value: p.value,
  }));

  return (
    <figure aria-label={`${metric} over time`} className="h-64 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
          <defs>
            <linearGradient id="seriesFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="var(--color-accent)" stopOpacity={0.35} />
              <stop offset="100%" stopColor="var(--color-accent)" stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid stroke="var(--color-separator)" strokeDasharray="3 3" vertical={false} />
          <XAxis
            dataKey="time"
            tick={{ fill: "var(--color-fg-subtle)", fontSize: 11 }}
            tickLine={false}
            axisLine={{ stroke: "var(--color-separator)" }}
            minTickGap={32}
          />
          <YAxis
            allowDecimals={false}
            width={36}
            tick={{ fill: "var(--color-fg-subtle)", fontSize: 11 }}
            tickLine={false}
            axisLine={false}
          />
          <ChartTooltip
            cursor={{ stroke: "var(--color-separator)" }}
            contentStyle={{
              background: "var(--color-elevated)",
              border: "1px solid var(--color-separator)",
              borderRadius: 8,
              color: "var(--color-fg)",
              fontSize: 12,
            }}
          />
          <Area
            type="monotone"
            dataKey="value"
            name={metric}
            stroke="var(--color-accent)"
            strokeWidth={2}
            fill="url(#seriesFill)"
          />
        </AreaChart>
      </ResponsiveContainer>
    </figure>
  );
}
