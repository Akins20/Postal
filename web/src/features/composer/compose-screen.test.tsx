import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it } from "vitest";

import { useWorkspaceStore } from "@/stores/workspace";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { ComposeScreen } from "./compose-screen";

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
// The LIST endpoint omits variants (backend `omitempty`); only the detail
// GET includes them — the mocks mirror that contract.
const LIST_POST = {
  id: "44444444-4444-4444-4444-444444444444",
  workspace_id: WS.id,
  status: "draft",
  created_at: "2026-01-02T00:00:00Z",
};
const DETAIL_POST = {
  ...LIST_POST,
  variants: [
    { id: "55555555-5555-5555-5555-555555555555", channel_id: CHANNEL.id, body: "Saved draft" },
  ],
};

function mockBase({ channels, posts }: { channels: unknown[]; posts: unknown[] }) {
  server.use(
    http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [WS] })),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/channels/`, () =>
      HttpResponse.json({ data: channels }),
    ),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/posts/`, () =>
      HttpResponse.json({ data: posts }),
    ),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/posts/${LIST_POST.id}`, () =>
      HttpResponse.json({ data: DETAIL_POST }),
    ),
  );
}

beforeEach(() => useWorkspaceStore.setState({ activeId: null }));

describe("ComposeScreen", () => {
  it("points to Channels when none are connected", async () => {
    mockBase({ channels: [], posts: [] });
    renderWithProviders(<ComposeScreen />);
    expect(await screen.findByText("Connect a channel first")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /go to channels/i })).toHaveAttribute(
      "href",
      "/channels",
    );
  });

  it("renders the composer and saved posts (list rows have no variants)", async () => {
    mockBase({ channels: [CHANNEL], posts: [LIST_POST] });
    renderWithProviders(<ComposeScreen />);
    expect(await screen.findByLabelText("Post text")).toBeInTheDocument();
    expect(await screen.findByText("Saved post")).toBeInTheDocument();
  });

  it("loads a draft into the composer via Edit (fetches the detail)", async () => {
    mockBase({ channels: [CHANNEL], posts: [LIST_POST] });
    renderWithProviders(<ComposeScreen />);
    await screen.findByText("Saved post");
    await userEvent.click(screen.getByRole("button", { name: "Edit" }));
    expect(await screen.findByText("Editing a saved draft.")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByLabelText("Post text")).toHaveValue("Saved draft"));
    expect(screen.getByRole("button", { name: "Update draft" })).toBeInTheDocument();
  });

  it("deletes a draft after confirmation", async () => {
    mockBase({ channels: [CHANNEL], posts: [LIST_POST] });
    let deleted = false;
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS.id}/posts/${LIST_POST.id}`, () => {
        deleted = true;
        return HttpResponse.json({ data: { message: "deleted" } });
      }),
    );
    renderWithProviders(<ComposeScreen />);
    await screen.findByText("Saved post");
    await userEvent.click(screen.getByRole("button", { name: "Delete" }));
    const dialog = await screen.findByRole("dialog", { name: "Delete this draft?" });
    await userEvent.click(within(dialog).getByRole("button", { name: "Delete" }));
    await waitFor(() => expect(deleted).toBe(true));
  });
});
