import { useQuery } from "@tanstack/react-query";

import { api, API_ORIGIN } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError } from "@/lib/api-error";

export type PostMetrics = components["schemas"]["PostMetrics"];
export type ChannelMetrics = components["schemas"]["ChannelMetrics"];
export type SeriesPoint = components["schemas"]["SeriesPoint"];

export const analyticsKeys = {
  overview: (workspaceId: string) => ["workspaces", workspaceId, "analytics"] as const,
  post: (workspaceId: string, postId: string) =>
    ["workspaces", workspaceId, "analytics", postId] as const,
  series: (workspaceId: string, postId: string, channelId: string, metric: string) =>
    ["workspaces", workspaceId, "analytics", postId, channelId, metric] as const,
};

/** Latest metrics per (post, channel) across the workspace. */
export function useAnalyticsOverview(workspaceId: string | undefined) {
  return useQuery<PostMetrics[]>({
    queryKey: analyticsKeys.overview(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: async () => {
      const { data, error, response } = await api.GET(
        "/api/v1/workspaces/{workspaceID}/analytics/",
        { params: { path: { workspaceID: workspaceId as string } } },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return (data.data.posts ?? []) as PostMetrics[];
    },
  });
}

/** Latest metrics for one post, broken out per channel. */
export function usePostAnalytics(workspaceId: string | undefined, postId: string | undefined) {
  return useQuery<ChannelMetrics[]>({
    queryKey: analyticsKeys.post(workspaceId ?? "", postId ?? ""),
    enabled: Boolean(workspaceId && postId),
    queryFn: async () => {
      const { data, error, response } = await api.GET(
        "/api/v1/workspaces/{workspaceID}/analytics/posts/{postID}",
        { params: { path: { workspaceID: workspaceId as string, postID: postId as string } } },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return (data.data.channels ?? []) as ChannelMetrics[];
    },
  });
}

/** Time series of one metric for a post on a channel (default window 30d). */
export function useMetricSeries(args: {
  workspaceId: string | undefined;
  postId: string | undefined;
  channelId: string | undefined;
  metric: string;
  from?: string;
  to?: string;
}) {
  const { workspaceId, postId, channelId, metric, from, to } = args;
  return useQuery<SeriesPoint[]>({
    queryKey: [
      ...analyticsKeys.series(workspaceId ?? "", postId ?? "", channelId ?? "", metric),
      { from, to },
    ],
    enabled: Boolean(workspaceId && postId && channelId && metric),
    queryFn: async () => {
      const { data, error, response } = await api.GET(
        "/api/v1/workspaces/{workspaceID}/analytics/posts/{postID}/series",
        {
          params: {
            path: { workspaceID: workspaceId as string, postID: postId as string },
            query: { channel_id: channelId as string, metric, from, to },
          },
        },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return (data.data.points ?? []) as SeriesPoint[];
    },
  });
}

/** Cookie-authenticated CSV export URL (opened via a plain link/anchor). */
export function analyticsCsvURL(workspaceId: string): string {
  return `${API_ORIGIN}/api/v1/workspaces/${workspaceId}/analytics/export.csv`;
}
