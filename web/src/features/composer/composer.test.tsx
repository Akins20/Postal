import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import type { Channel } from "@/data/channels";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { Composer } from "./composer";

const WS_ID = "11111111-1111-1111-1111-111111111111";
const CH_A: Channel = {
  id: "22222222-2222-2222-2222-222222222222",
  platform: "twitter",
  platform_account_id: "1",
  handle: "ada",
  display_name: "Ada",
  status: "active",
  created_at: "2026-01-01T00:00:00Z",
};
const CH_B: Channel = { ...CH_A, id: "33333333-3333-3333-3333-333333333333", handle: "grace" };
const POST = {
  id: "44444444-4444-4444-4444-444444444444",
  workspace_id: WS_ID,
  status: "draft",
  created_at: "2026-01-01T00:00:00Z",
  variants: [{ id: "55555555-5555-5555-5555-555555555555", channel_id: CH_A.id, body: "Hello" }],
};

describe("Composer", () => {
  it("disables save until a channel is picked and text entered", async () => {
    renderWithProviders(<Composer workspaceId={WS_ID} channels={[CH_A]} />);
    const save = screen.getByRole("button", { name: "Save draft" });
    expect(save).toBeDisabled();
    await userEvent.click(screen.getByRole("checkbox", { name: "@ada" }));
    expect(save).toBeDisabled();
    await userEvent.type(screen.getByLabelText("Post text"), "Hello world");
    expect(save).toBeEnabled();
  });

  it("counts characters against the platform limit", async () => {
    renderWithProviders(<Composer workspaceId={WS_ID} channels={[CH_A]} />);
    await userEvent.click(screen.getByRole("checkbox", { name: "@ada" }));
    await userEvent.type(screen.getByLabelText("Post text"), "12345");
    expect(screen.getByText("275 left")).toBeInTheDocument();
  });

  it("saves a draft, validates it, and shows per-channel verdicts", async () => {
    let sent: { variants: { channel_id: string; body: string }[] } | null = null;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/`, async ({ request }) => {
        sent = (await request.json()) as typeof sent;
        return HttpResponse.json({ data: POST }, { status: 201 });
      }),
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/${POST.id}/validate`, () =>
        HttpResponse.json({ data: { variants: [{ channel_id: CH_A.id, valid: true }] } }),
      ),
    );
    renderWithProviders(<Composer workspaceId={WS_ID} channels={[CH_A]} />);
    await userEvent.click(screen.getByRole("checkbox", { name: "@ada" }));
    await userEvent.type(screen.getByLabelText("Post text"), "Hello");
    await userEvent.click(screen.getByRole("button", { name: "Save draft" }));
    expect(await screen.findByText(/draft saved/i)).toBeInTheDocument();
    expect(screen.getByText("Ready")).toBeInTheDocument();
    await waitFor(() => expect(sent?.variants).toEqual([{ channel_id: CH_A.id, body: "Hello" }]));
  });

  it("shows an invalid verdict with the server message", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/`, () =>
        HttpResponse.json({ data: POST }, { status: 201 }),
      ),
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/${POST.id}/validate`, () =>
        HttpResponse.json({
          data: {
            variants: [
              { channel_id: CH_A.id, valid: false, code: "too_long", message: "exceeds 280" },
            ],
          },
        }),
      ),
    );
    renderWithProviders(<Composer workspaceId={WS_ID} channels={[CH_A]} />);
    await userEvent.click(screen.getByRole("checkbox", { name: "@ada" }));
    await userEvent.type(screen.getByLabelText("Post text"), "Hello");
    await userEvent.click(screen.getByRole("button", { name: "Save draft" }));
    expect(await screen.findByText("Needs changes")).toBeInTheDocument();
    expect(screen.getByText(/exceeds 280/)).toBeInTheDocument();
  });

  it("sends a per-channel override from its tab", async () => {
    let sent: { variants: { channel_id: string; body: string }[] } | null = null;
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/`, async ({ request }) => {
        sent = (await request.json()) as typeof sent;
        return HttpResponse.json({ data: POST }, { status: 201 });
      }),
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/${POST.id}/validate`, () =>
        HttpResponse.json({ data: { variants: [] } }),
      ),
    );
    renderWithProviders(<Composer workspaceId={WS_ID} channels={[CH_A, CH_B]} />);
    await userEvent.click(screen.getByRole("checkbox", { name: "@ada" }));
    await userEvent.click(screen.getByRole("checkbox", { name: "@grace" }));
    await userEvent.type(screen.getByLabelText("Post text"), "Master");
    // Tabs appear once 2+ channels are selected.
    await userEvent.click(screen.getByRole("tab", { name: "@grace" }));
    const graceEditor = screen.getByLabelText("Text for @grace");
    await userEvent.clear(graceEditor);
    await userEvent.type(graceEditor, "Custom for grace");
    await userEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() =>
      expect(sent?.variants).toEqual([
        { channel_id: CH_A.id, body: "Master" },
        { channel_id: CH_B.id, body: "Custom for grace" },
      ]),
    );
  });

  it("attaches library media and sends it with the variants", async () => {
    const ASSET = {
      id: "66666666-6666-6666-6666-666666666666",
      workspace_id: WS_ID,
      kind: "image",
      mime: "image/png",
      width: 10,
      height: 10,
      duration_ms: 0,
      bytes: 2048,
      status: "uploaded",
      created_at: "2026-01-01T00:00:00Z",
    };
    let sent: { variants: { media?: { media_id: string }[] }[] } | null = null;
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/media/`, () =>
        HttpResponse.json({ data: [ASSET] }),
      ),
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/`, async ({ request }) => {
        sent = (await request.json()) as typeof sent;
        return HttpResponse.json({ data: POST }, { status: 201 });
      }),
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/${POST.id}/validate`, () =>
        HttpResponse.json({ data: { variants: [] } }),
      ),
    );
    renderWithProviders(<Composer workspaceId={WS_ID} channels={[CH_A]} />);
    await userEvent.click(screen.getByRole("checkbox", { name: "@ada" }));
    await userEvent.type(screen.getByLabelText("Post text"), "With media");
    await userEvent.click(screen.getByRole("button", { name: /attach media/i }));
    await userEvent.click(await screen.findByRole("button", { name: /image \(image\/png\)/i }));
    expect(screen.getByText(/image · 2\.0 KiB/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() =>
      expect(sent?.variants[0].media).toEqual([
        { media_id: ASSET.id, kind: "image", mime: "image/png", bytes: 2048 },
      ]),
    );
  });

  it("surfaces a save failure", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/`, () =>
        HttpResponse.json(
          { error: { code: "validation", message: "body too long" } },
          { status: 400 },
        ),
      ),
    );
    renderWithProviders(<Composer workspaceId={WS_ID} channels={[CH_A]} />);
    await userEvent.click(screen.getByRole("checkbox", { name: "@ada" }));
    await userEvent.type(screen.getByLabelText("Post text"), "Hello");
    await userEvent.click(screen.getByRole("button", { name: "Save draft" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("body too long");
  });
});
