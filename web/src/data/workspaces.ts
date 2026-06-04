import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import type { Capability, Role } from "@/config/capabilities";
import { normalizeError, type NormalizedError } from "@/lib/api-error";

export type Workspace = components["schemas"]["Workspace"];
export type Member = components["schemas"]["Member"];

export const wsKeys = {
  list: ["workspaces"] as const,
  members: (id: string) => ["workspaces", id, "members"] as const,
};

/** Workspaces the signed-in user belongs to. */
export function useWorkspaces() {
  return useQuery<Workspace[]>({
    queryKey: wsKeys.list,
    queryFn: async () => {
      const { data, error, response } = await api.GET("/api/v1/workspaces/");
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return (data.data ?? []) as Workspace[];
    },
  });
}

export function useMembers(workspaceId: string | undefined) {
  return useQuery<Member[]>({
    queryKey: wsKeys.members(workspaceId ?? ""),
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

interface AddMemberInput {
  email: string;
  role?: Role;
  capabilities?: Capability[];
}

export function useAddMember(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<Member, NormalizedError, AddMemberInput>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST("/api/v1/workspaces/{workspaceID}/members", {
        params: { path: { workspaceID: workspaceId } },
        body,
      });
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return data.data as Member;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: wsKeys.members(workspaceId) }),
  });
}

interface UpdateCapsInput {
  userId: string;
  role?: Role;
  capabilities?: Capability[];
}

export function useUpdateCapabilities(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<Member, NormalizedError, UpdateCapsInput>({
    mutationFn: async ({ userId, ...body }) => {
      const { data, error, response } = await api.PATCH(
        "/api/v1/workspaces/{workspaceID}/members/{userID}/capabilities",
        { params: { path: { workspaceID: workspaceId, userID: userId } }, body },
      );
      if (!response.ok || !data) throw normalizeError(response.status, error);
      return data.data as Member;
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: wsKeys.members(workspaceId) }),
  });
}
