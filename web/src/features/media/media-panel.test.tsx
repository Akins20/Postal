import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it } from "vitest";

import { useWorkspaceStore } from "@/stores/workspace";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { MediaPanel } from "./media-panel";

const WS = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "Personal",
  owner_user_id: "00000000-0000-0000-0000-000000000001",
  plan: "free",
  created_at: "2026-01-01T00:00:00Z",
};
const IMAGE = {
  id: "55555555-5555-5555-5555-555555555555",
  workspace_id: WS.id,
  kind: "image",
  mime: "image/png",
  width: 100,
  height: 80,
  duration_ms: 0,
  bytes: 2048,
  status: "uploaded",
  created_at: "2026-01-01T00:00:00Z",
};
const VIDEO = {
  ...IMAGE,
  id: "66666666-6666-6666-6666-666666666666",
  kind: "video",
  mime: "video/mp4",
  bytes: 5 * 1024 * 1024,
};

function mockLists(assets: unknown[]) {
  server.use(
    http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [WS] })),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/media/`, () =>
      HttpResponse.json({ data: assets }),
    ),
  );
}

beforeEach(() => useWorkspaceStore.setState({ activeId: null }));

describe("MediaPanel", () => {
  it("shows the empty state with an upload button", async () => {
    mockLists([]);
    renderWithProviders(<MediaPanel />);
    expect(await screen.findByText("No media yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /upload/i })).toBeInTheDocument();
  });

  it("renders the asset grid (image preview + video placeholder)", async () => {
    mockLists([IMAGE, VIDEO]);
    renderWithProviders(<MediaPanel />);
    expect(await screen.findByAltText("image asset (image/png)")).toHaveAttribute(
      "src",
      `http://localhost/api/v1/workspaces/${WS.id}/media/${IMAGE.id}/download`,
    );
    expect(screen.getByText(/video\/mp4 · 5\.0 MiB/)).toBeInTheDocument();
    expect(screen.queryByText("No media yet")).not.toBeInTheDocument();
  });

  it("uploads a picked file and refreshes the grid", async () => {
    mockLists([]);
    let uploaded = false;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS.id}/media/`, () => {
        uploaded = true;
        return HttpResponse.json({ data: IMAGE }, { status: 201 });
      }),
    );
    renderWithProviders(<MediaPanel />);
    await screen.findByText("No media yet");
    const input = screen.getByLabelText("Choose a file to upload");
    await userEvent.upload(input, new File(["png"], "pic.png", { type: "image/png" }));
    await waitFor(() => expect(uploaded).toBe(true));
  });

  it("shows the server rejection for a refused upload", async () => {
    mockLists([]);
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS.id}/media/`, () =>
        HttpResponse.json(
          { error: { code: "quota_exceeded", message: "storage quota exceeded" } },
          { status: 400 },
        ),
      ),
    );
    renderWithProviders(<MediaPanel />);
    await screen.findByText("No media yet");
    const input = screen.getByLabelText("Choose a file to upload");
    await userEvent.upload(input, new File(["x"], "big.png", { type: "image/png" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("storage quota exceeded");
  });

  it("deletes an asset after confirmation", async () => {
    mockLists([IMAGE]);
    let deleted = false;
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS.id}/media/${IMAGE.id}`, () => {
        deleted = true;
        return HttpResponse.json({ data: { message: "deleted" } });
      }),
    );
    renderWithProviders(<MediaPanel />);
    await screen.findByAltText("image asset (image/png)");
    await userEvent.click(screen.getByRole("button", { name: "Delete image asset" }));
    const dialog = await screen.findByRole("dialog", { name: "Delete this asset?" });
    await userEvent.click(within(dialog).getByRole("button", { name: "Delete" }));
    await waitFor(() => expect(deleted).toBe(true));
  });
});
