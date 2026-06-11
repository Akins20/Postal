import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";

export type Channel = components["schemas"]["Channel"];
export type ChannelStatus = Channel["status"];

export const channelKeys = {
  list: (workspaceId: string) => ["workspaces", workspaceId, "channels"] as const,
};

/** Connected social accounts for a workspace. */
export function useChannels(workspaceId: string | undefined) {
  return useQuery<Channel[]>({
    queryKey: channelKeys.list(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: async () => {
      const { data, error, response } = await api.GET(
        "/api/v1/workspaces/{workspaceID}/channels/",
        { params: { path: { workspaceID: workspaceId as string } } },
      );
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return (data.data ?? []) as Channel[];
    },
  });
}

/**
 * Begin the OAuth connect flow. Returns the IdP authorize URL; the caller
 * redirects the browser there and the IdP sends the user back to our
 * `/oauth/callback` route.
 */
export function useConnectChannel(workspaceId: string) {
  return useMutation<string, NormalizedError, { platform: string }>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST(
        "/api/v1/workspaces/{workspaceID}/channels/connect",
        { params: { path: { workspaceID: workspaceId } }, body },
      );
      if (!response.ok || !data?.data?.authorize_url) {
        throw normalizeError(response.status, error);
      }
      return data.data.authorize_url;
    },
  });
}

/**
 * Complete the OAuth flow with the state+code the IdP appended to our
 * callback URL. The backend re-validates the single-use state and returns
 * the connected channel.
 */
export function useCompleteOAuth() {
  const qc = useQueryClient();
  return useMutation<Channel, NormalizedError, { state: string; code: string }>({
    mutationFn: async (query) => {
      const { data, error, response } = await api.GET("/api/v1/channels/oauth/callback", {
        params: { query },
      });
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return data.data as Channel;
    },
    // The state is workspace-bound server-side; we don't know which workspace
    // it belongs to here, so refresh every channels list we hold.
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workspaces"] }),
  });
}

export function useDisconnectChannel(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<void, NormalizedError, { channelId: string }>({
    mutationFn: async ({ channelId }) => {
      const { data, error, response } = await api.DELETE(
        "/api/v1/workspaces/{workspaceID}/channels/{channelID}",
        { params: { path: { workspaceID: workspaceId, channelID: channelId } } },
      );
      if (!response.ok || !data) throw normalizeError(response.status, error);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: channelKeys.list(workspaceId) }),
  });
}
