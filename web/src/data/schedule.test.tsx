import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/react";

import {
  useCalendar,
  useCancelJob,
  useCreateSlot,
  useDeleteSlot,
  useSchedulePost,
  useSlots,
} from "./schedule";

const WS_ID = "11111111-1111-1111-1111-111111111111";
const CH_ID = "22222222-2222-2222-2222-222222222222";
const POST_ID = "33333333-3333-3333-3333-333333333333";
const JOB = {
  id: "44444444-4444-4444-4444-444444444444",
  post_id: POST_ID,
  channel_id: CH_ID,
  run_at: "2026-06-15T09:00:00Z",
  status: "scheduled",
  attempts: 0,
  created_at: "2026-06-11T00:00:00Z",
};
const SLOT = {
  id: "55555555-5555-5555-5555-555555555555",
  channel_id: CH_ID,
  day_of_week: 1,
  time_of_day: "09:00",
  timezone: "America/New_York",
  created_at: "2026-06-11T00:00:00Z",
};

describe("useSchedulePost", () => {
  it("schedules at a specific time and returns the jobs", async () => {
    let sent: Record<string, unknown> | null = null;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/schedule`, async ({ request }) => {
        sent = (await request.json()) as typeof sent;
        return HttpResponse.json({ data: { jobs: [JOB] } }, { status: 201 });
      }),
    );
    const { result } = renderHook(() => useSchedulePost(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ postId: POST_ID, runAt: JOB.run_at });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].status).toBe("scheduled");
    await waitFor(() => expect(sent).toMatchObject({ post_id: POST_ID, run_at: JOB.run_at }));
  });

  it("schedules into the next open slots", async () => {
    let sent: Record<string, unknown> | null = null;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/schedule`, async ({ request }) => {
        sent = (await request.json()) as typeof sent;
        return HttpResponse.json({ data: { jobs: [JOB] } }, { status: 201 });
      }),
    );
    const { result } = renderHook(() => useSchedulePost(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ postId: POST_ID, toSlots: true });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    await waitFor(() => expect(sent).toMatchObject({ post_id: POST_ID, to_slots: true }));
  });

  it("surfaces a scheduling rejection (e.g. past time)", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/schedule`, () =>
        HttpResponse.json(
          { error: { code: "validation", message: "run_at must be in the future" } },
          { status: 400 },
        ),
      ),
    );
    const { result } = renderHook(() => useSchedulePost(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ postId: POST_ID, runAt: "2020-01-01T00:00:00Z" });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("run_at must be in the future");
  });
});

describe("useCalendar", () => {
  it("stays idle without a workspace id", () => {
    const { result } = renderHook(() => useCalendar(undefined), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("lists jobs in a window", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/calendar`, ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("from")).toBe("2026-06-01T00:00:00Z");
        return HttpResponse.json({ data: { jobs: [JOB] } });
      }),
    );
    const { result } = renderHook(
      () => useCalendar(WS_ID, "2026-06-01T00:00:00Z", "2026-07-01T00:00:00Z"),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].id).toBe(JOB.id);
  });
});

describe("useCancelJob", () => {
  it("cancels a job", async () => {
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS_ID}/scheduled-jobs/${JOB.id}`, () =>
        HttpResponse.json({ data: { message: "canceled" } }),
      ),
    );
    const { result } = renderHook(() => useCancelJob(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ jobId: JOB.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("surfaces not-found for an already-run job", async () => {
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS_ID}/scheduled-jobs/${JOB.id}`, () =>
        HttpResponse.json(
          { error: { code: "not_found", message: "job not found" } },
          { status: 404 },
        ),
      ),
    );
    const { result } = renderHook(() => useCancelJob(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ jobId: JOB.id });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.status).toBe(404);
  });
});

describe("useSlots", () => {
  it("stays idle without a channel id", () => {
    const { result } = renderHook(() => useSlots(WS_ID, undefined), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("lists a channel's slots", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/slots/`, ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("channel_id")).toBe(CH_ID);
        return HttpResponse.json({ data: [SLOT] });
      }),
    );
    const { result } = renderHook(() => useSlots(WS_ID, CH_ID), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].time_of_day).toBe("09:00");
  });
});

describe("useCreateSlot / useDeleteSlot", () => {
  it("creates a slot", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/slots/`, () =>
        HttpResponse.json({ data: SLOT }, { status: 201 }),
      ),
    );
    const { result } = renderHook(() => useCreateSlot(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({
      channel_id: CH_ID,
      day_of_week: 1,
      time_of_day: "09:00",
      timezone: "America/New_York",
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.id).toBe(SLOT.id);
  });

  it("deletes a slot (channel_id goes along as a query param)", async () => {
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS_ID}/slots/${SLOT.id}`, ({ request }) => {
        const url = new URL(request.url);
        expect(url.searchParams.get("channel_id")).toBe(CH_ID);
        return HttpResponse.json({ data: { message: "deleted" } });
      }),
    );
    const { result } = renderHook(() => useDeleteSlot(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ slotId: SLOT.id, channelId: CH_ID });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});
