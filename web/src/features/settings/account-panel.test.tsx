import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import type { Workspace } from "@/data/workspaces";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { AccountPanel } from "./account-panel";

const WS: Workspace = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "Personal",
  owner_user_id: "00000000-0000-0000-0000-000000000001",
  plan: "free",
  created_at: "2026-01-01T00:00:00Z",
};

describe("AccountPanel", () => {
  it("shows the signed-in account with verification state and workspace facts", async () => {
    server.use(
      http.get("http://localhost/api/v1/auth/me", () =>
        HttpResponse.json({
          data: {
            id: "00000000-0000-0000-0000-000000000001",
            email: "ada@example.com",
            email_verified: true,
            status: "active",
            created_at: "2026-02-01T00:00:00Z",
          },
        }),
      ),
    );
    renderWithProviders(<AccountPanel workspace={WS} />);
    expect(await screen.findByText("ada@example.com")).toBeInTheDocument();
    expect(screen.getByText("Verified")).toBeInTheDocument();
    expect(screen.getByText("1 February 2026")).toBeInTheDocument();
    expect(screen.getByText("Personal")).toBeInTheDocument();
    expect(screen.getByText("free")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Toggle color theme" })).toBeInTheDocument();
  });

  it("flags an unverified email", async () => {
    server.use(
      http.get("http://localhost/api/v1/auth/me", () =>
        HttpResponse.json({
          data: {
            id: "00000000-0000-0000-0000-000000000001",
            email: "ada@example.com",
            email_verified: false,
            status: "active",
            created_at: "2026-02-01T00:00:00Z",
          },
        }),
      ),
    );
    renderWithProviders(<AccountPanel workspace={WS} />);
    expect(await screen.findByText("Unverified")).toBeInTheDocument();
  });
});
