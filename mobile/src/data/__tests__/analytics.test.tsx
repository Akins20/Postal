import { renderHook, waitFor } from "@testing-library/react-native";

import { useAnalyticsOverview, usePostAnalytics } from "@/data/analytics";
import { useLedger, useWallet } from "@/data/billing";
import { mockRoute } from "@/test/fetch-mock";
import { createWrapper } from "@/test/react";

const WS = "11111111-1111-1111-1111-111111111111";
const ROW = { post_id: "p1", channel_id: "c1", platform_post_id: "190", metrics: { likes: 12, reposts: 3 }, captured_at: "2026-06-10T00:00:00Z" };

describe("useAnalyticsOverview", () => {
  it("lists latest metrics per post/channel", async () => {
    mockRoute("GET", `/workspaces/${WS}/analytics/`, 200, { data: { posts: [ROW] } });
    const { result } = await renderHook(() => useAnalyticsOverview(WS), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].metrics.likes).toBe(12);
  });
});

describe("usePostAnalytics", () => {
  it("breaks a post out per channel", async () => {
    mockRoute("GET", `/workspaces/${WS}/analytics/posts/p1`, 200, {
      data: { post_id: "p1", channels: [{ channel_id: "c1", platform_post_id: "190", metrics: { likes: 12 }, captured_at: ROW.captured_at }] },
    });
    const { result } = await renderHook(() => usePostAnalytics(WS, "p1"), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].channel_id).toBe("c1");
  });
});

describe("billing reads", () => {
  it("loads the wallet with tier prices", async () => {
    mockRoute("GET", `/workspaces/${WS}/billing/wallet`, 200, {
      data: { workspace_id: WS, balance: 975, publish_costs: { twitter: 10, twitter_media: 15, twitter_url: 25 }, updated_at: "2026-06-12T00:00:00Z" },
    });
    const { result } = await renderHook(() => useWallet(WS), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.balance).toBe(975);
    expect(result.current.data?.publish_costs.twitter_url).toBe(25);
  });

  it("lists the ledger", async () => {
    mockRoute("GET", `/workspaces/${WS}/billing/ledger`, 200, {
      data: [{ id: "l1", workspace_id: WS, kind: "topup", credits: 1000, reference: "r", note: "n", created_at: "2026-06-12T00:00:00Z" }],
    });
    const { result } = await renderHook(() => useLedger(WS), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].kind).toBe("topup");
  });
});
