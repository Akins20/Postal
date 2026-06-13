import { useQuery } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError } from "@/lib/api-error";

export type Workspace = components["schemas"]["Workspace"];

export const wsKeys = { list: ["workspaces"] as const };

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
