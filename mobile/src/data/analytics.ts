import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError } from "@/lib/api-error";

export type PostMetrics = components["schemas"]["PostMetrics"];
export type ChannelMetrics = components["schemas"]["ChannelMetrics"];

export const analyticsKeys = {
  overview: (workspaceId: string) => ["workspaces", workspaceId, "analytics"] as const,
  post: (workspaceId: string, postId: string) =>
    ["workspaces", workspaceId, "analytics", postId] as const,
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
