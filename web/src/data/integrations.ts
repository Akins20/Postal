import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";

export type Integration = components["schemas"]["Integration"];

export const integrationKeys = {
  list: (workspaceId: string) => ["workspaces", workspaceId, "integrations"] as const,
};

/** The workspace's integrations (secrets never leave the server). */
export function useIntegrations(workspaceId: string | undefined) {
  return useQuery<Integration[]>({
    queryKey: integrationKeys.list(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: async () => {
      const { data, error, response } = await api.GET(
        "/api/v1/workspaces/{workspaceID}/integrations/",
        { params: { path: { workspaceID: workspaceId as string } } },
      );
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return (data.data ?? []) as Integration[];
    },
  });
}

export interface ConfigureIntegrationInput {
  provider: "ogshortener";
  enabled: boolean;
  autoApply?: boolean;
  /** Verified by the provider, then encrypted at rest; omit to keep the stored key. */
  apiKey?: string;
}

export function useConfigureIntegration(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<Integration, NormalizedError, ConfigureIntegrationInput>({
    mutationFn: async ({ provider, enabled, autoApply, apiKey }) => {
      const { data, error, response } = await api.PUT(
        "/api/v1/workspaces/{workspaceID}/integrations/{provider}",
        {
          params: { path: { workspaceID: workspaceId, provider } },
          body: { enabled, auto_apply: autoApply ?? false, api_key: apiKey ?? null },
        },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return data.data as Integration;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: integrationKeys.list(workspaceId) }),
  });
}

/** Replace every link in text with an OGShortener short link. */
export function useShortenLinks(workspaceId: string) {
  return useMutation<string, NormalizedError, { text: string }>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST(
        "/api/v1/workspaces/{workspaceID}/integrations/ogshortener/shorten",
        { params: { path: { workspaceID: workspaceId } }, body },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return data.data.text ?? "";
    },
  });
}
