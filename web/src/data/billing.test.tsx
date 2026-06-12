import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/react";

import { useLedger, useTopup, useWallet } from "./billing";

const WS_ID = "11111111-1111-1111-1111-111111111111";
const WALLET = {
  workspace_id: WS_ID,
  balance: 975,
  publish_costs: { twitter: 25 },
  updated_at: "2026-06-12T00:00:00Z",
};
const ENTRY = {
  id: "22222222-2222-2222-2222-222222222222",
  workspace_id: WS_ID,
  kind: "topup",
  credits: 1000,
  reference: "stripe:evt_1",
  note: "wallet top-up",
  created_at: "2026-06-12T00:00:00Z",
};

describe("useWallet", () => {
  it("stays idle without a workspace id", () => {
    const { result } = renderHook(() => useWallet(undefined), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("returns the balance and price list", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/billing/wallet`, () =>
        HttpResponse.json({ data: WALLET }),
      ),
    );
    const { result } = renderHook(() => useWallet(WS_ID), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.balance).toBe(975);
    expect(result.current.data?.publish_costs.twitter).toBe(25);
  });
});

describe("useLedger", () => {
  it("lists movements", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/billing/ledger`, () =>
        HttpResponse.json({ data: [ENTRY] }),
      ),
    );
    const { result } = renderHook(() => useLedger(WS_ID), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].kind).toBe("topup");
  });
});

describe("useTopup", () => {
  it("returns the hosted checkout URL", async () => {
    let sent: Record<string, unknown> | null = null;
    server.use(
      http.post(
        `http://localhost/api/v1/workspaces/${WS_ID}/billing/topup`,
        async ({ request }) => {
          sent = (await request.json()) as typeof sent;
          return HttpResponse.json({ data: { checkout_url: "https://pay.test/cs_1" } });
        },
      ),
    );
    const { result } = renderHook(() => useTopup(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ provider: "stripe", credits: 1000 });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBe("https://pay.test/cs_1");
    await waitFor(() => expect(sent).toMatchObject({ provider: "stripe", credits: 1000 }));
  });

  it("surfaces below-minimum and missing-permission errors", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/billing/topup`, () =>
        HttpResponse.json(
          { error: { code: "invalid_topup", message: "minimum top-up is 500 credits" } },
          { status: 400 },
        ),
      ),
    );
    const { result } = renderHook(() => useTopup(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ provider: "stripe", credits: 1 });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("minimum top-up is 500 credits");
  });
});
