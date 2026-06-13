import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { API_ORIGIN, refreshSession } from "@/api/client";
import { normalizeError, type NormalizedError } from "@/lib/api-error";
import { getAccessToken } from "@/lib/secure-session";

/**
 * Governance hooks (per-channel publish permissions + activity feed). These
 * endpoints are not in the generated schema, so they use a direct Bearer fetch
 * with the same refresh-on-401 behaviour as the typed client.
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

async function authedFetch(path: string, init?: RequestInit): Promise<Response> {
  const run = () => {
    const token = getAccessToken();
    return fetch(`${API_ORIGIN}${path}`, {
      ...init,
      headers: {
        ...(init?.headers ?? {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    });
  };
  let res = await run();
  if (res.status === 401 && (await refreshSession())) res = await run();
  return res;
}

async function authedJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await authedFetch(path, init);
  const body = await res.json().catch(() => undefined);
  if (!res.ok) throw normalizeError(res.status, body);
  return (body?.data ?? body) as T;
}

export const governanceKeys = {
  channels: (ws: string, user: string) => ["workspaces", ws, "member-channels", user] as const,
  activity: (ws: string) => ["workspaces", ws, "activity"] as const,
};

export function useMemberChannels(workspaceId: string, userId: string) {
  return useQuery<ChannelAccess>({
    queryKey: governanceKeys.channels(workspaceId, userId),
    enabled: Boolean(workspaceId && userId),
    queryFn: () =>
      authedJSON<ChannelAccess>(`/api/v1/workspaces/${workspaceId}/members/${userId}/channels`),
  });
}

export function useSetMemberChannels(workspaceId: string, userId: string) {
  const qc = useQueryClient();
  return useMutation<ChannelAccess, NormalizedError, { restricted: boolean; channel_ids: string[] }>({
    mutationFn: (body) =>
      authedJSON<ChannelAccess>(`/api/v1/workspaces/${workspaceId}/members/${userId}/channels`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: governanceKeys.channels(workspaceId, userId) }),
  });
}

export function useActivity(workspaceId: string | undefined) {
  return useQuery<ActivityEntry[]>({
    queryKey: governanceKeys.activity(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: () => authedJSON<ActivityEntry[]>(`/api/v1/workspaces/${workspaceId}/activity`),
  });
}
