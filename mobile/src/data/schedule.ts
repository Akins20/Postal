import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "@/api/client";
import type { components } from "@/api/schema";
import { normalizeError, type NormalizedError } from "@/lib/api-error";

export type Job = components["schemas"]["Job"];
export type JobStatus = Job["status"];
export type Slot = components["schemas"]["Slot"];
export type SlotRequest = components["schemas"]["SlotRequest"];

export const scheduleKeys = {
  calendar: (workspaceId: string) => ["workspaces", workspaceId, "calendar"] as const,
  slots: (workspaceId: string, channelId: string) =>
    ["workspaces", workspaceId, "slots", channelId] as const,
};

export interface ScheduleInput {
  postId: string;
  /** Exact publish time (ISO). Mutually exclusive with toSlots. */
  runAt?: string;
  /** Queue each variant into its channel's next open posting slot. */
  toSlots?: boolean;
}

/** Schedule a saved post: one job per variant/channel. */
export function useSchedulePost(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<Job[], NormalizedError, ScheduleInput>({
    mutationFn: async ({ postId, runAt, toSlots }) => {
      const { data, error, response } = await api.POST(
        "/api/v1/workspaces/{workspaceID}/schedule",
        {
          params: { path: { workspaceID: workspaceId } },
          body: { post_id: postId, run_at: runAt, to_slots: toSlots },
        },
      );
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return (data.data.jobs ?? []) as Job[];
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["workspaces", workspaceId, "calendar"] });
      qc.invalidateQueries({ queryKey: ["workspaces", workspaceId, "posts"] });
    },
  });
}

/** Scheduled jobs in a window (backend default [now, now+30d)). */
export function useCalendar(workspaceId: string | undefined, from?: string, to?: string) {
  return useQuery<Job[]>({
    queryKey: [...scheduleKeys.calendar(workspaceId ?? ""), { from, to }],
    enabled: Boolean(workspaceId),
    queryFn: async () => {
      const { data, error, response } = await api.GET("/api/v1/workspaces/{workspaceID}/calendar", {
        params: { path: { workspaceID: workspaceId as string }, query: { from, to } },
      });
      if (!response.ok || !data?.data) throw normalizeError(response.status, error);
      return (data.data.jobs ?? []) as Job[];
    },
  });
}

export function useCancelJob(workspaceId: string) {
  const qc = useQueryClient();
  return useMutation<void, NormalizedError, { jobId: string }>({
    mutationFn: async ({ jobId }) => {
      const { data, error, response } = await api.DELETE(
        "/api/v1/workspaces/{workspaceID}/scheduled-jobs/{jobID}",
        { params: { path: { workspaceID: workspaceId, jobID: jobId } } },
      );
      if (!response.ok || !data) throw normalizeError(response.status, error);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workspaces", workspaceId, "calendar"] }),
  });
}
