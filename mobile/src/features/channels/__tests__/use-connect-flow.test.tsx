import { renderHook } from "@testing-library/react-native";

import { useConnectFlow } from "@/features/channels/use-connect-flow";
import { mockRoute } from "@/test/fetch-mock";
import { createWrapper } from "@/test/react";

const mockOpenAuth = jest.fn();
jest.mock("expo-web-browser", () => ({
  openAuthSessionAsync: (...args: unknown[]) => mockOpenAuth(...args),
}));
jest.mock("expo-linking", () => ({
  parse: (url: string) => {
    const q = url.split("?")[1] ?? "";
    const queryParams: Record<string, string> = {};
    for (const pair of q.split("&")) {
      const [k, v] = pair.split("=");
      if (k) queryParams[k] = decodeURIComponent(v ?? "");
    }
    return { queryParams };
  },
}));

const WS = "11111111-1111-1111-1111-111111111111";
const CHANNEL = {
  id: "c1", platform: "instagram", platform_account_id: "1",
  handle: "simgram", display_name: "Sim Gram", status: "active",
  connected_by: null, created_at: "2026-01-01T00:00:00Z",
};

beforeEach(() => mockOpenAuth.mockReset());

describe("useConnectFlow", () => {
  it("opens the authorize URL and completes with state+code from the redirect", async () => {
    mockRoute("POST", `/workspaces/${WS}/channels/connect`, 200, {
      data: { authorize_url: "https://idp.test/auth?x=1" },
    });
    mockRoute("GET", "/channels/oauth/callback", 201, { data: CHANNEL });
    mockOpenAuth.mockResolvedValue({
      type: "success",
      url: "postal://oauth-callback?state=st1&code=cd1",
    });

    const { result } = await renderHook(() => useConnectFlow(WS), { wrapper: createWrapper() });
    const res = await result.current.run("instagram");

    expect(mockOpenAuth).toHaveBeenCalledWith("https://idp.test/auth?x=1", "postal://oauth-callback");
    expect(res).toEqual({ status: "connected", channel: CHANNEL });
  });

  it("reports cancellation when the user dismisses the browser", async () => {
    mockRoute("POST", `/workspaces/${WS}/channels/connect`, 200, {
      data: { authorize_url: "https://idp.test/auth" },
    });
    mockOpenAuth.mockResolvedValue({ type: "cancel" });

    const { result } = await renderHook(() => useConnectFlow(WS), { wrapper: createWrapper() });
    expect(await result.current.run("twitter")).toEqual({ status: "cancelled" });
  });

  it("surfaces a server error from the exchange", async () => {
    mockRoute("POST", `/workspaces/${WS}/channels/connect`, 200, {
      data: { authorize_url: "https://idp.test/auth" },
    });
    mockRoute("GET", "/channels/oauth/callback", 400, {
      error: { code: "invalid_state", message: "link expired" },
    });
    mockOpenAuth.mockResolvedValue({
      type: "success",
      url: "postal://oauth-callback?state=stale&code=c",
    });

    const { result } = await renderHook(() => useConnectFlow(WS), { wrapper: createWrapper() });
    expect(await result.current.run("twitter")).toEqual({ status: "error", message: "link expired" });
  });
});
