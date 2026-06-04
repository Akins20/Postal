import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it, vi } from "vitest";

import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { AuthGuard } from "./auth-guard";

const { replace } = vi.hoisted(() => ({ replace: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));

const USER = {
  id: "00000000-0000-0000-0000-000000000001",
  email: "ada@example.com",
  email_verified: true,
  status: "active",
  created_at: "2026-01-01T00:00:00Z",
};

describe("AuthGuard", () => {
  it("renders children when signed in", async () => {
    server.use(
      http.get("http://localhost/api/v1/auth/me", () => HttpResponse.json({ data: USER })),
    );
    renderWithProviders(
      <AuthGuard>
        <p>protected content</p>
      </AuthGuard>,
    );
    expect(await screen.findByText("protected content")).toBeInTheDocument();
  });

  it("redirects to /login when not signed in", async () => {
    server.use(
      http.get("http://localhost/api/v1/auth/me", () => new HttpResponse(null, { status: 401 })),
      http.post(
        "http://localhost/api/v1/auth/refresh",
        () => new HttpResponse(null, { status: 401 }),
      ),
    );
    renderWithProviders(
      <AuthGuard>
        <p>protected content</p>
      </AuthGuard>,
    );
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/login"));
    expect(screen.queryByText("protected content")).not.toBeInTheDocument();
  });
});
