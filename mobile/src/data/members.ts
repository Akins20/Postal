import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";

export type Member = components["schemas"]["Member"];
export type Role = "owner" | "admin" | "editor" | "viewer" | "custom";
// Roles an owner can assign (owner is implicit, custom is not directly picked).
export type AssignableRole = "admin" | "editor" | "viewer";

export const ASSIGNABLE_ROLES: AssignableRole[] = ["admin", "editor", "viewer"];
export const ROLE_LABELS: Record<string, string> = {
  owner: "Owner",
  admin: "Admin",
  editor: "Editor",
  viewer: "Viewer",
  custom: "Custom",
};

export const memberKeys = {
  list: (workspaceId: string) => ["workspaces", workspaceId, "members"] as const,
};

/** Members of a workspace. */
export function useMembers(workspaceId: string | undefined) {
  return useQuery<Member[]>({
    queryKey: memberKeys.list(workspaceId ?? ""),
    enabled: Boolean(workspaceId),
    queryFn: async () => {
      const { data, error, response } = await api.GET("/api/v1/workspaces/{workspaceID}/members", {
        params: { path: { workspaceID: workspaceId as string } },
      });
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return (data.data ?? []) as Member[];
    },
  });
}

/** Add a member to the workspace by email with a starting role. */
export function useAddMember(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<Member, NormalizedError, { email: string; role: AssignableRole }>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST("/api/v1/workspaces/{workspaceID}/members", {
        params: { path: { workspaceID: workspaceId } },
        body,
      });
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return data.data as Member;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: memberKeys.list(workspaceId) }),
  });
}
