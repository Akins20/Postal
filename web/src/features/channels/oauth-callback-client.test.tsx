import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { OAuthCallbackClient } from "./oauth-callback-client";

const { replace } = vi.hoisted(() => ({ replace: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));

const CHANNEL = {
  id: "22222222-2222-2222-2222-222222222222",
  platform: "twitter",
  platform_account_id: "1",
  handle: "ada",
  display_name: "Ada Lovelace",
  status: "active",
  connected_by: null,
  created_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => replace.mockClear());

describe("OAuthCallbackClient", () => {
  it("completes the exchange once and redirects to /channels", async () => {
    let calls = 0;
    server.use(
      http.get("http://localhost/api/v1/channels/oauth/callback", () => {
        calls += 1;
        return HttpResponse.json({ data: CHANNEL }, { status: 201 });
      }),
    );
    renderWithProviders(<OAuthCallbackClient state="s1" code="c1" />);
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/channels"));
    expect(calls).toBe(1);
  });

  it("shows the backend error and a way back", async () => {
    server.use(
      http.get("http://localhost/api/v1/channels/oauth/callback", () =>
        HttpResponse.json(
          { error: { code: "validation", message: "state expired" } },
          { status: 400 },
        ),
      ),
    );
    renderWithProviders(<OAuthCallbackClient state="stale" code="c" />);
    expect(await screen.findByText("state expired")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /back to channels/i })).toBeInTheDocument();
    expect(replace).not.toHaveBeenCalled();
  });

  it("fails fast when state or code is missing", async () => {
    renderWithProviders(<OAuthCallbackClient />);
    expect(await screen.findByText(/missing its authorization details/i)).toBeInTheDocument();
    expect(replace).not.toHaveBeenCalled();
  });
});
