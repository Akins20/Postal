import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { normalizeError, type NormalizedError } from "@/lib/api-error";

/**
 * Governance hooks (per-channel publish permissions + the activity feed). These
 * endpoints are not in the generated OpenAPI schema, so they use direct fetch
 * through the same-origin /api proxy.
 */

export interface ChannelAccess {
  restricted: boolean;
  allowed_channel_ids: string[];
}

export interface ActivityEntry {
  id: number;
  actor_email: string;
  action: string;
  target: string;
  created_at: string;
}

async function getJSON<T>(url: string): Promise<T> {
  const res = await fetch(url, { credentials: "include" });
  const body = await res.json().catch(() => undefined);
  if (!res.ok) throw normalizeError(res.status, body);
  return (body?.data ?? body) as T;
}

export const governanceKeys = {
  channels: (ws: string, user: string) => ["workspaces", ws, "member-channels", user] as const,
  activity: (ws: string) => ["workspaces", ws, "activity"] as const,
};

/** A member's per-channel publish access. */
export function useMemberChannels(workspaceId: string, userId: string) {
  return useQuery<ChannelAccess>({
    queryKey: governanceKeys.channels(workspaceId, userId),
    enabled: Boolean(workspaceId && userId),
    queryFn: () =>
      getJSON<ChannelAccess>(`/api/v1/workspaces/${workspaceId}/members/${userId}/channels`),
  });
}

/** Replace a member's per-channel publish allowlist. */
export function useSetMemberChannels(workspaceId: string, userId: string) {
  const qc = useQueryClient();
  return useMutation<
    ChannelAccess,
    NormalizedError,
    { restricted: boolean; channel_ids: string[] }
  >({
    mutationFn: async (body) => {
      const csrf = (await import("@/api/client")).csrfToken();
      const res = await fetch(`/api/v1/workspaces/${workspaceId}/members/${userId}/channels`, {
        method: "PUT",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          ...(csrf ? { "X-CSRF-Token": csrf } : {}),
        },
        body: JSON.stringify(body),
      });
      const data = await res.json().catch(() => undefined);
      if (!res.ok) throw normalizeError(res.status, data);
      return (data?.data ?? data) as ChannelAccess;
    },
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: governanceKeys.channels(workspaceId, userId) }),
  });
}

/** Recent workspace activity (who did what). */
export function useActivity(workspaceId: string | undefined) {
  return useQuery<ActivityEntry[]>({
    queryKey: governanceKeys.activity(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: () => getJSON<ActivityEntry[]>(`/api/v1/workspaces/${workspaceId}/activity`),
  });
}
