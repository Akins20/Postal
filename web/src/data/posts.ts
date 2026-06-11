import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";

export type Post = components["schemas"]["Post"];
export type Variant = components["schemas"]["Variant"];
export type VariantInput = components["schemas"]["VariantInput"];
export type MediaMeta = components["schemas"]["MediaMeta"];
export type VariantValidation = components["schemas"]["VariantValidation"];

export const postKeys = {
  list: (workspaceId: string) => ["workspaces", workspaceId, "posts"] as const,
  detail: (workspaceId: string, postId: string) =>
    ["workspaces", workspaceId, "posts", postId] as const,
};

/** Posts (drafts and published) in a workspace. */
export function usePosts(workspaceId: string | undefined) {
  return useQuery<Post[]>({
    queryKey: postKeys.list(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: async () => {
      const { data, error, response } = await api.GET("/api/v1/workspaces/{workspaceID}/posts/", {
        params: { path: { workspaceID: workspaceId as string } },
      });
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return (data.data ?? []) as Post[];
    },
  });
}

export function usePost(workspaceId: string | undefined, postId: string | undefined) {
  return useQuery<Post>({
    queryKey: postKeys.detail(workspaceId ?? "", postId ?? ""),
    enabled: Boolean(workspaceId && postId),
    queryFn: async () => {
      const { data, error, response } = await api.GET(
        "/api/v1/workspaces/{workspaceID}/posts/{postID}",
        { params: { path: { workspaceID: workspaceId as string, postID: postId as string } } },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return data.data as Post;
    },
  });
}

export function useCreatePost(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<Post, NormalizedError, { variants: VariantInput[] }>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST("/api/v1/workspaces/{workspaceID}/posts/", {
        params: { path: { workspaceID: workspaceId } },
        body,
      });
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return data.data as Post;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: postKeys.list(workspaceId) }),
  });
}

export function useUpdatePost(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<Post, NormalizedError, { postId: string; variants: VariantInput[] }>({
    mutationFn: async ({ postId, ...body }) => {
      const { data, error, response } = await api.PUT(
        "/api/v1/workspaces/{workspaceID}/posts/{postID}",
        { params: { path: { workspaceID: workspaceId, postID: postId } }, body },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return data.data as Post;
    },
    onSuccess: (post) => {
      qc.invalidateQueries({ queryKey: postKeys.list(workspaceId) });
      qc.setQueryData(postKeys.detail(workspaceId, post.id), post);
    },
  });
}

export function useDeletePost(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<void, NormalizedError, { postId: string }>({
    mutationFn: async ({ postId }) => {
      const { data, error, response } = await api.DELETE(
        "/api/v1/workspaces/{workspaceID}/posts/{postID}",
        { params: { path: { workspaceID: workspaceId, postID: postId } } },
      );
      if (!response.ok || !data) throw normalizeError(response.status, error);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: postKeys.list(workspaceId) }),
  });
}

/** Server-side validation of a saved post's variants against their platforms. */
export function useValidatePost(workspaceId: string) {
  return useMutation<VariantValidation[], NormalizedError, { postId: string }>({
    mutationFn: async ({ postId }) => {
      const { data, error, response } = await api.POST(
        "/api/v1/workspaces/{workspaceID}/posts/{postID}/validate",
        { params: { path: { workspaceID: workspaceId, postID: postId } } },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return (data.data.variants ?? []) as VariantValidation[];
    },
  });
}

/** Preview how links in `text` look once workspace UTM params are applied. */
export function useUtmPreview(workspaceId: string) {
  return useMutation<string, NormalizedError, { text: string; utm?: Record<string, string> }>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST(
        "/api/v1/workspaces/{workspaceID}/posts/utm-preview",
        { params: { path: { workspaceID: workspaceId } }, body },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return data.data.text ?? "";
    },
  });
}
