import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, API_ORIGIN, refreshSession } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";
import { getAccessToken } from "@/lib/secure-session";

export type Asset = components["schemas"]["Asset"];

/** A picked file ready to upload (expo-image-picker shape). */
export interface PickedFile {
  uri: string;
  name: string;
  mime: string;
}

export const mediaKeys = {
  list: (workspaceId: string) => ["workspaces", workspaceId, "media"] as const,
};

/** Media assets uploaded to a workspace. */
export function useMedia(workspaceId: string | undefined) {
  return useQuery<Asset[]>({
    queryKey: mediaKeys.list(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: async () => {
      const { data, error, response } = await api.GET("/api/v1/workspaces/{workspaceID}/media/", {
        params: { path: { workspaceID: workspaceId as string } },
      });
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return (data.data ?? []) as Asset[];
    },
  });
}

/** Bytes URL + auth headers for an asset (feed both to <Image source>). */
export function mediaSource(workspaceId: string, mediaId: string): { uri: string; headers: Record<string, string> } {
  const token = getAccessToken();
  return {
    uri: `${API_ORIGIN}/api/v1/workspaces/${workspaceId}/media/${mediaId}/download`,
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  };
}

/**
 * Upload one picked file as multipart/form-data via React Native's FormData
 * (file objects carry a local uri). Manual fetch so we control the Bearer
 * header and a single refresh-on-401 retry, mirroring the api client.
 */
export async function uploadMedia(workspaceId: string, file: PickedFile): Promise<Asset> {
  const url = `${API_ORIGIN}/api/v1/workspaces/${workspaceId}/media/`;
  const send = async () => {
    const form = new FormData();
    // RN FormData accepts a file descriptor object for multipart.
    form.append("file", { uri: file.uri, name: file.name, type: file.mime } as unknown as Blob);
    const token = getAccessToken();
    return fetch(url, {
      method: "POST",
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    });
  };
  let res = await send();
  if (res.status === 401 && (await refreshSession())) res = await send();
  const body = (await res.json().catch(() => null)) as { data?: Asset } | null;
  if (res.status !== 201 || !body?.data) throw normalizeError(res.status, body);
  return body.data;
}

export function useUploadMedia(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<Asset, NormalizedError, PickedFile>({
    mutationFn: (file) => uploadMedia(workspaceId, file),
    onSuccess: () => qc.invalidateQueries({ queryKey: mediaKeys.list(workspaceId) }),
  });
}

export function useDeleteMedia(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<void, NormalizedError, { mediaId: string }>({
    mutationFn: async ({ mediaId }) => {
      const { data, error, response } = await api.DELETE(
        "/api/v1/workspaces/{workspaceID}/media/{mediaID}",
        { params: { path: { workspaceID: workspaceId, mediaID: mediaId } } },
      );
      if (!response.ok || !data) throw normalizeError(response.status, error);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: mediaKeys.list(workspaceId) }),
  });
}
