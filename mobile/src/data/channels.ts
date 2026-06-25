import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, API_ORIGIN, refreshSession } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";
import { getAccessToken } from "@/lib/secure-session";

export type Channel = components["schemas"]["Channel"];
export type ChannelStatus = Channel["status"];

/** Deep link the IdP redirects back to (app.json scheme "postal"). The OAuth
 *  redirect allowlist on the server must include this exact value. */
export const OAUTH_REDIRECT = "postal://oauth-callback";

export const channelKeys = {
  list: (workspaceId: string) =>
    ["workspaces", workspaceId, "channels"] as const,
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
 * Begin the OAuth connect flow. Returns the IdP authorize URL, built server-
 * side to redirect to OUR app deep link (allowlisted), so the in-app browser
 * hands control back to the app.
 */
export function useConnectChannel(workspaceId: string) {
  return useMutation<string, NormalizedError, { platform: string }>({
    mutationFn: async ({ platform }) => {
      const { data, error, response } = await api.POST(
        "/api/v1/workspaces/{workspaceID}/channels/connect",
        {
          params: { path: { workspaceID: workspaceId } },
          body: { platform, redirect_uri: OAUTH_REDIRECT },
        },
      );
      if (!response.ok || !data?.data?.authorize_url)
        throw normalizeError(response.status, error);
      return data.data.authorize_url;
    },
  });
}

/** Complete the OAuth flow with the state+code from the redirect. */
export function useCompleteOAuth() {
  const qc = useQueryClient();
  return useMutation<Channel, NormalizedError, { state: string; code: string }>(
    {
      mutationFn: async (query) => {
        const { data, error, response } = await api.GET(
          "/api/v1/channels/oauth/callback",
          {
            params: { query },
          },
        );
        if (!response.ok || !data?.data)
          throw normalizeError(response.status, error);
        return data.data as Channel;
      },
      onSuccess: () => qc.invalidateQueries({ queryKey: ["workspaces"] }),
    },
  );
}

/**
 * Connect a manual (non-OAuth) provider like Telegram from supplied credentials.
 * Direct Bearer fetch (endpoint not in the schema) with refresh-on-401.
 */
export function useConnectManual(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<
    void,
    NormalizedError,
    { platform: string; credentials: Record<string, string> }
  >({
    mutationFn: async (body) => {
      const run = () => {
        const token = getAccessToken();
        return fetch(
          `${API_ORIGIN}/api/v1/workspaces/${workspaceId}/channels/connect-manual`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              ...(token ? { Authorization: `Bearer ${token}` } : {}),
            },
            body: JSON.stringify(body),
          },
        );
      };
      let res = await run();
      if (res.status === 401 && (await refreshSession())) res = await run();
      if (!res.ok) {
        const err = await res.json().catch(() => undefined);
        throw normalizeError(res.status, err);
      }
    },
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: channelKeys.list(workspaceId) }),
  });
}

export function useDisconnectChannel(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<void, NormalizedError, { channelId: string }>({
    mutationFn: async ({ channelId }) => {
      const { data, error, response } = await api.DELETE(
        "/api/v1/workspaces/{workspaceID}/channels/{channelID}",
        {
          params: { path: { workspaceID: workspaceId, channelID: channelId } },
        },
      );
      if (!response.ok || !data) throw normalizeError(response.status, error);
    },
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: channelKeys.list(workspaceId) }),
  });
}
