import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it } from "vitest";

import { useWorkspaceStore } from "@/stores/workspace";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { ChannelsPanel } from "./channels-panel";

const WS = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "Personal",
  owner_user_id: "00000000-0000-0000-0000-000000000001",
  plan: "free",
  created_at: "2026-01-01T00:00:00Z",
};
const ACTIVE = {
  id: "22222222-2222-2222-2222-222222222222",
  platform: "twitter",
  platform_account_id: "1",
  handle: "ada",
  display_name: "Ada Lovelace",
  status: "active",
  connected_by: null,
  created_at: "2026-01-01T00:00:00Z",
};
const EXPIRED = {
  ...ACTIVE,
  id: "33333333-3333-3333-3333-333333333333",
  handle: "grace",
  display_name: "Grace Hopper",
  status: "expired",
};

function mockLists(channels: unknown[]) {
  server.use(
    http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [WS] })),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/channels/`, () =>
      HttpResponse.json({ data: channels }),
    ),
  );
}

beforeEach(() => useWorkspaceStore.setState({ activeId: null }));

describe("ChannelsPanel", () => {
  it("shows the empty state and the connect list when nothing is connected", async () => {
    mockLists([]);
    renderWithProviders(<ChannelsPanel />);
    expect(await screen.findByText("No accounts connected yet")).toBeInTheDocument();
    expect(screen.getByText("X (Twitter)")).toBeInTheDocument();
    expect(screen.getByText("Instagram")).toBeInTheDocument();
    expect(screen.getByText("TikTok")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Connect" })).toHaveLength(3);
  });

  it("lists connected channels with their health status", async () => {
    mockLists([ACTIVE, EXPIRED]);
    renderWithProviders(<ChannelsPanel />);
    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.getByText("Grace Hopper")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Expired")).toBeInTheDocument();
    expect(screen.queryByText("No accounts connected yet")).not.toBeInTheDocument();
  });

  it("disconnects a channel after confirmation", async () => {
    mockLists([ACTIVE]);
    let deleted = false;
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS.id}/channels/${ACTIVE.id}`, () => {
        deleted = true;
        return HttpResponse.json({ data: { message: "disconnected" } });
      }),
    );
    renderWithProviders(<ChannelsPanel />);
    await screen.findByText("Ada Lovelace");
    await userEvent.click(screen.getByRole("button", { name: "Disconnect" }));
    const dialog = await screen.findByRole("dialog", { name: "Disconnect @ada?" });
    await userEvent.click(within(dialog).getByRole("button", { name: "Disconnect" }));
    await waitFor(() => expect(deleted).toBe(true));
  });

  it("surfaces a load error", async () => {
    server.use(
      http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [WS] })),
      http.get(`http://localhost/api/v1/workspaces/${WS.id}/channels/`, () =>
        HttpResponse.json({ error: { code: "forbidden", message: "nope" } }, { status: 403 }),
      ),
    );
    renderWithProviders(<ChannelsPanel />);
    expect(await screen.findByRole("alert")).toHaveTextContent(/couldn't load channels/i);
  });
});
