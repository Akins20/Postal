import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it } from "vitest";

import { useWorkspaceStore } from "@/stores/workspace";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { AnalyticsScreen } from "./analytics-screen";

const WS = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "Personal",
  owner_user_id: "00000000-0000-0000-0000-000000000001",
  plan: "free",
  created_at: "2026-01-01T00:00:00Z",
};
const CHANNEL = {
  id: "22222222-2222-2222-2222-222222222222",
  platform: "twitter",
  platform_account_id: "1",
  handle: "ada",
  display_name: "Ada",
  status: "active",
  connected_by: null,
  created_at: "2026-01-01T00:00:00Z",
};
const POST = {
  id: "33333333-3333-3333-3333-333333333333",
  workspace_id: WS.id,
  status: "published",
  created_at: "2026-06-01T00:00:00Z",
  variants: [
    { id: "55555555-5555-5555-5555-555555555555", channel_id: CHANNEL.id, body: "Launch day!" },
  ],
};
const ROW = {
  post_id: POST.id,
  channel_id: CHANNEL.id,
  platform_post_id: "190000000",
  metrics: { likes: 12, reposts: 3 },
  captured_at: "2026-06-10T08:00:00Z",
};

function mockBase(rows: unknown[]) {
  server.use(
    http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [WS] })),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/channels/`, () =>
      HttpResponse.json({ data: [CHANNEL] }),
    ),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/posts/`, () =>
      HttpResponse.json({ data: [POST] }),
    ),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/analytics/`, () =>
      HttpResponse.json({ data: { posts: rows } }),
    ),
  );
}

beforeEach(() => useWorkspaceStore.setState({ activeId: null }));

describe("AnalyticsScreen", () => {
  it("explains the wait when posts are published but unpolled", async () => {
    mockBase([]);
    renderWithProviders(<AnalyticsScreen />);
    expect(await screen.findByText("Metrics are on the way")).toBeInTheDocument();
  });

  it("lists overview rows with post excerpt, channel and metrics", async () => {
    mockBase([ROW]);
    renderWithProviders(<AnalyticsScreen />);
    expect(await screen.findByText("Launch day!")).toBeInTheDocument();
    expect(screen.getByText("@ada")).toBeInTheDocument();
    expect(screen.getByText("likes 12 · reposts 3")).toBeInTheDocument();
  });

  it("links the cookie-authenticated CSV export", async () => {
    mockBase([ROW]);
    renderWithProviders(<AnalyticsScreen />);
    await screen.findByText("Launch day!");
    expect(screen.getByRole("link", { name: /export csv/i })).toHaveAttribute(
      "href",
      `http://localhost/api/v1/workspaces/${WS.id}/analytics/export.csv`,
    );
  });

  it("drills into a post: per-channel numbers and a metric series", async () => {
    mockBase([ROW]);
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS.id}/analytics/posts/${POST.id}`, () =>
        HttpResponse.json({
          data: {
            post_id: POST.id,
            channels: [
              {
                channel_id: CHANNEL.id,
                platform_post_id: "190000000",
                metrics: { likes: 12, reposts: 3 },
                captured_at: ROW.captured_at,
              },
            ],
          },
        }),
      ),
      http.get(
        `http://localhost/api/v1/workspaces/${WS.id}/analytics/posts/${POST.id}/series`,
        ({ request }) => {
          const url = new URL(request.url);
          expect(url.searchParams.get("metric")).toBe("likes");
          return HttpResponse.json({
            data: {
              post_id: POST.id,
              channel_id: CHANNEL.id,
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
    renderWithProviders(<AnalyticsScreen />);
    await userEvent.click(await screen.findByText("Launch day!"));
    expect(await screen.findByText("Post breakdown")).toBeInTheDocument();
    // Per-channel latest numbers (dt/dd cards).
    expect((await screen.findAllByText("likes")).length).toBeGreaterThan(0);
    expect(screen.getByText("12")).toBeInTheDocument();
    // The series chart mounts with the default metric.
    expect(await screen.findByRole("figure", { name: "likes over time" })).toBeInTheDocument();
  });
});
