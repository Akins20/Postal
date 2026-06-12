import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, API_ORIGIN, csrfToken } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";

export type Asset = components["schemas"]["Asset"];

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

/** Cookie-authenticated URL serving an asset's bytes (usable as an img/video src). */
export function mediaDownloadURL(workspaceId: string, mediaId: string): string {
  return `${API_ORIGIN}/api/v1/workspaces/${workspaceId}/media/${mediaId}/download`;
}

/**
 * Upload one file as multipart/form-data via XHR - fetch can't report upload
 * progress. Mirrors the api client's conventions: cookie credentials, CSRF
 * double-submit, request-id correlation.
 */
export function uploadMedia(
  workspaceId: string,
  file: File,
  onProgress?: (fraction: number) => void,
): Promise<Asset> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", `${API_ORIGIN}/api/v1/workspaces/${workspaceId}/media/`);
    xhr.withCredentials = true;
    const csrf = csrfToken();
    if (csrf) xhr.setRequestHeader("X-CSRF-Token", csrf);
    xhr.setRequestHeader("X-Request-Id", crypto.randomUUID());
    xhr.responseType = "json";

    xhr.upload.addEventListener("progress", (e) => {
      if (e.lengthComputable && onProgress) onProgress(e.loaded / e.total);
    });
    xhr.addEventListener("load", () => {
      const body = xhr.response as { data?: Asset } | null;
      if (xhr.status === 201 && body?.data) resolve(body.data);
      else reject(normalizeError(xhr.status, body));
    });
    xhr.addEventListener("error", () => reject(normalizeError(0, null)));
    xhr.addEventListener("abort", () => reject(normalizeError(0, null)));

    const form = new FormData();
    form.append("file", file);
    xhr.send(form);
  });
}

export interface UploadInput {
  file: File;
  onProgress?: (fraction: number) => void;
}

export function useUploadMedia(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<Asset, NormalizedError, UploadInput>({
    mutationFn: ({ file, onProgress }) => uploadMedia(workspaceId, file, onProgress),
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
