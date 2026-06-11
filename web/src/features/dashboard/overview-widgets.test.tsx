import { screen } from "@testing-library/react";
import { addDays } from "date-fns";
import { http, HttpResponse } from "msw";
import { axe } from "vitest-axe";
import { beforeEach, describe, expect, it } from "vitest";

import { useWorkspaceStore } from "@/stores/workspace";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { OverviewWidgets } from "./overview-widgets";

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
  status: "expired",
  connected_by: null,
  created_at: "2026-01-01T00:00:00Z",
};
const JOB = {
  id: "44444444-4444-4444-4444-444444444444",
  post_id: "33333333-3333-3333-3333-333333333333",
  channel_id: CHANNEL.id,
  run_at: addDays(new Date(), 2).toISOString(),
  status: "scheduled",
  attempts: 0,
  created_at: "2026-01-01T00:00:00Z",
};
const DRAFT = {
  id: "33333333-3333-3333-3333-333333333333",
  workspace_id: WS.id,
  status: "draft",
  created_at: "2026-06-01T00:00:00Z",
  variants: [
    { id: "55555555-5555-5555-5555-555555555555", channel_id: CHANNEL.id, body: "WIP idea" },
  ],
};

function mockAll({
  jobs,
  posts,
  channels,
}: {
  jobs: unknown[];
  posts: unknown[];
  channels: unknown[];
}) {
  server.use(
    http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [WS] })),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/calendar`, () =>
      HttpResponse.json({ data: { jobs } }),
    ),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/posts/`, () =>
      HttpResponse.json({ data: posts }),
    ),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/channels/`, () =>
      HttpResponse.json({ data: channels }),
    ),
  );
}

beforeEach(() => useWorkspaceStore.setState({ activeId: null }));

describe("OverviewWidgets", () => {
  it("shows upcoming jobs, drafts, and channel health", async () => {
    mockAll({ jobs: [JOB], posts: [DRAFT], channels: [CHANNEL] });
    renderWithProviders(<OverviewWidgets />);
    // "@ada" shows in both the schedule widget and the channel-health list.
    expect(await screen.findAllByText("@ada")).toHaveLength(2);
    expect(await screen.findByText("WIP idea")).toBeInTheDocument();
    expect(await screen.findByText(/1 connected/)).toBeInTheDocument();
    expect(screen.getByText(/1 need attention/)).toBeInTheDocument();
  });

  it("shows friendly empties when there's nothing yet", async () => {
    mockAll({ jobs: [], posts: [], channels: [] });
    renderWithProviders(<OverviewWidgets />);
    expect(await screen.findByText("Nothing scheduled yet.")).toBeInTheDocument();
    expect(screen.getByText("No drafts — write something.")).toBeInTheDocument();
    expect(screen.getByText("No accounts connected yet.")).toBeInTheDocument();
  });

  it("has no axe violations", async () => {
    mockAll({ jobs: [JOB], posts: [DRAFT], channels: [CHANNEL] });
    const { container } = renderWithProviders(<OverviewWidgets />);
    await screen.findAllByText("@ada");
    expect(await axe(container)).toHaveNoViolations();
  });
});
