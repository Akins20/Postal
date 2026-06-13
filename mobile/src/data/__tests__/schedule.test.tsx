import { renderHook, waitFor } from "@testing-library/react-native";

import { useCalendar, useCancelJob, useSchedulePost } from "@/data/schedule";
import { calls, mockRoute } from "@/test/fetch-mock";
import { createWrapper } from "@/test/react";

const WS = "11111111-1111-1111-1111-111111111111";
const JOB = {
  id: "44444444-4444-4444-4444-444444444444",
  post_id: "p1",
  channel_id: "c1",
  run_at: "2026-06-20T09:00:00Z",
  status: "scheduled",
  attempts: 0,
  created_at: "2026-06-13T00:00:00Z",
};

describe("useSchedulePost", () => {
  it("schedules at a specific time", async () => {
    mockRoute("POST", `/workspaces/${WS}/schedule`, 201, { data: { jobs: [JOB] } });
    const { result } = await renderHook(() => useSchedulePost(WS), { wrapper: createWrapper() });
    result.current.mutate({ postId: "p1", runAt: JOB.run_at });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].status).toBe("scheduled");
    const call = calls.find((c) => c.url.includes("/schedule"));
    expect(call?.body).toMatchObject({ post_id: "p1", run_at: JOB.run_at });
  });

  it("schedules into next open slots", async () => {
    mockRoute("POST", `/workspaces/${WS}/schedule`, 201, { data: { jobs: [JOB, { ...JOB, id: "x" }] } });
    const { result } = await renderHook(() => useSchedulePost(WS), { wrapper: createWrapper() });
    result.current.mutate({ postId: "p1", toSlots: true });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(2);
    const call = calls.find((c) => c.url.includes("/schedule"));
    expect(call?.body).toMatchObject({ post_id: "p1", to_slots: true });
  });

  it("surfaces insufficient credits", async () => {
    mockRoute("POST", `/workspaces/${WS}/schedule`, 400, {
      error: { code: "insufficient_credits", message: "not enough wallet credits" },
    });
    const { result } = await renderHook(() => useSchedulePost(WS), { wrapper: createWrapper() });
    result.current.mutate({ postId: "p1", toSlots: true });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.code).toBe("insufficient_credits");
  });
});

describe("useCalendar", () => {
  it("lists jobs in the window", async () => {
    mockRoute("GET", `/workspaces/${WS}/calendar`, 200, { data: { jobs: [JOB] } });
    const { result } = await renderHook(
      () => useCalendar(WS, "2026-06-01T00:00:00Z", "2026-07-01T00:00:00Z"),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].id).toBe(JOB.id);
  });
});

describe("useCancelJob", () => {
  it("cancels a job", async () => {
    mockRoute("DELETE", `/workspaces/${WS}/scheduled-jobs/${JOB.id}`, 200, { data: { message: "ok" } });
    const { result } = await renderHook(() => useCancelJob(WS), { wrapper: createWrapper() });
    result.current.mutate({ jobId: JOB.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});
