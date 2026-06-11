import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/react";

import {
  analyticsCsvURL,
  useAnalyticsOverview,
  useMetricSeries,
  usePostAnalytics,
} from "./analytics";

const WS_ID = "11111111-1111-1111-1111-111111111111";
const POST_ID = "33333333-3333-3333-3333-333333333333";
const CH_ID = "22222222-2222-2222-2222-222222222222";
const ROW = {
  post_id: POST_ID,
  channel_id: CH_ID,
  platform_post_id: "190000000",
  metrics: { likes: 12, reposts: 3 },
  captured_at: "2026-06-10T00:00:00Z",
};

describe("useAnalyticsOverview", () => {
  it("stays idle without a workspace id", () => {
    const { result } = renderHook(() => useAnalyticsOverview(undefined), {
      wrapper: createWrapper(),
    });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("lists latest metrics per post/channel", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/analytics/`, () =>
        HttpResponse.json({ data: { posts: [ROW] } }),
      ),
    );
    const { result } = renderHook(() => useAnalyticsOverview(WS_ID), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].metrics.likes).toBe(12);
  });

  it("surfaces a forbidden error", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/analytics/`, () =>
        HttpResponse.json({ error: { code: "forbidden", message: "nope" } }, { status: 403 }),
      ),
    );
    const { result } = renderHook(() => useAnalyticsOverview(WS_ID), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("usePostAnalytics", () => {
  it("breaks a post out per channel", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/analytics/posts/${POST_ID}`, () =>
        HttpResponse.json({
          data: {
            post_id: POST_ID,
            channels: [
              {
                channel_id: CH_ID,
                platform_post_id: "190000000",
                metrics: { likes: 12 },
                captured_at: ROW.captured_at,
              },
            ],
          },
        }),
      ),
    );
    const { result } = renderHook(() => usePostAnalytics(WS_ID, POST_ID), {
      wrapper: createWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].channel_id).toBe(CH_ID);
  });
});

describe("useMetricSeries", () => {
  it("fetches points for a metric with the channel as a query param", async () => {
    server.use(
      http.get(
        `http://localhost/api/v1/workspaces/${WS_ID}/analytics/posts/${POST_ID}/series`,
        ({ request }) => {
          const url = new URL(request.url);
          expect(url.searchParams.get("channel_id")).toBe(CH_ID);
          expect(url.searchParams.get("metric")).toBe("likes");
          return HttpResponse.json({
            data: {
              post_id: POST_ID,
              channel_id: CH_ID,
              metric: "likes",
              points: [
                { value: 5, captured_at: "2026-06-09T00:00:00Z" },
                { value: 12, captured_at: "2026-06-10T00:00:00Z" },
              ],
            },
          });
        },
      ),
    );
    const { result } = renderHook(
      () =>
        useMetricSeries({ workspaceId: WS_ID, postId: POST_ID, channelId: CH_ID, metric: "likes" }),
      { wrapper: createWrapper() },
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.map((p) => p.value)).toEqual([5, 12]);
  });

  it("stays idle until a channel is chosen", () => {
    const { result } = renderHook(
      () =>
        useMetricSeries({
          workspaceId: WS_ID,
          postId: POST_ID,
          channelId: undefined,
          metric: "likes",
        }),
      { wrapper: createWrapper() },
    );
    expect(result.current.fetchStatus).toBe("idle");
  });
});

describe("analyticsCsvURL", () => {
  it("builds the export URL", () => {
    expect(analyticsCsvURL(WS_ID)).toBe(
      `http://localhost/api/v1/workspaces/${WS_ID}/analytics/export.csv`,
    );
  });
});
