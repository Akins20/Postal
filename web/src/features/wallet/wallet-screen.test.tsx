import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useWorkspaceStore } from "@/stores/workspace";
import { server } from "@/test/msw/server";
import { renderWithProviders } from "@/test/react";

import { WalletScreen } from "./wallet-screen";

const { params } = vi.hoisted(() => ({ params: new URLSearchParams() }));
vi.mock("next/navigation", () => ({ useSearchParams: () => params }));

const WS = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "Personal",
  owner_user_id: "00000000-0000-0000-0000-000000000001",
  plan: "free",
  created_at: "2026-01-01T00:00:00Z",
};
const WALLET = {
  workspace_id: WS.id,
  balance: 975,
  publish_costs: { twitter: 25 },
  updated_at: "2026-06-12T00:00:00Z",
};
const LEDGER = [
  {
    id: "22222222-2222-2222-2222-222222222222",
    workspace_id: WS.id,
    kind: "topup",
    credits: 1000,
    reference: "stripe:evt_1",
    note: "wallet top-up",
    created_at: "2026-06-12T00:00:00Z",
  },
  {
    id: "33333333-3333-3333-3333-333333333333",
    workspace_id: WS.id,
    kind: "publish_charge",
    credits: -25,
    reference: "job-1",
    note: "publish to twitter",
    created_at: "2026-06-12T01:00:00Z",
  },
];

function mockBase() {
  server.use(
    http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [WS] })),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/billing/wallet`, () =>
      HttpResponse.json({ data: WALLET }),
    ),
    http.get(`http://localhost/api/v1/workspaces/${WS.id}/billing/ledger`, () =>
      HttpResponse.json({ data: LEDGER }),
    ),
  );
}

beforeEach(() => useWorkspaceStore.setState({ activeId: null }));
afterEach(() => vi.unstubAllGlobals());

describe("WalletScreen", () => {
  it("shows the balance, X price, and history", async () => {
    mockBase();
    renderWithProviders(<WalletScreen />);
    expect(await screen.findByText("975")).toBeInTheDocument();
    expect(screen.getByText("credits per X post")).toBeInTheDocument();
    expect(screen.getByText("Top-up")).toBeInTheDocument();
    expect(screen.getByText("X publish")).toBeInTheDocument();
    expect(screen.getByText("-25")).toBeInTheDocument();
  });

  it("starts a checkout and redirects the browser to it", async () => {
    mockBase();
    const assign = vi.fn();
    vi.stubGlobal("location", { ...window.location, assign });
    let sent: Record<string, unknown> | null = null;
    server.use(
      http.post(
        `http://localhost/api/v1/workspaces/${WS.id}/billing/topup`,
        async ({ request }) => {
          sent = (await request.json()) as typeof sent;
          return HttpResponse.json({ data: { checkout_url: "https://pay.test/cs_1" } });
        },
      ),
    );
    renderWithProviders(<WalletScreen />);
    await screen.findByText("975");
    // $25 (USD via Stripe) maps to 2500 credits.
    const amount = screen.getByLabelText("Amount (USD)");
    await userEvent.clear(amount);
    await userEvent.type(amount, "25");
    await userEvent.click(screen.getByRole("button", { name: /buy 2500 credits/i }));
    await waitFor(() => expect(assign).toHaveBeenCalledWith("https://pay.test/cs_1"));
    await waitFor(() => expect(sent).toMatchObject({ provider: "stripe", credits: 2500 }));
  });

  it("surfaces a refused top-up", async () => {
    mockBase();
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS.id}/billing/topup`, () =>
        HttpResponse.json(
          { error: { code: "forbidden", message: "You don't have permission to do that." } },
          { status: 403 },
        ),
      ),
    );
    renderWithProviders(<WalletScreen />);
    await screen.findByText("975");
    // Default $5 = 500 credits.
    await userEvent.click(screen.getByRole("button", { name: /buy 500 credits/i }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/permission/i);
  });
});
