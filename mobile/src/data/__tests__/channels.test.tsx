import { renderHook, waitFor } from "@testing-library/react-native";

import {
  OAUTH_REDIRECT,
  useChannels,
  useCompleteOAuth,
  useConnectChannel,
  useDisconnectChannel,
} from "@/data/channels";
import { calls, mockRoute } from "@/test/fetch-mock";
import { createWrapper } from "@/test/react";

const WS = "11111111-1111-1111-1111-111111111111";
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

describe("useChannels", () => {
  it("stays idle without a workspace id", async () => {
    const { result } = await renderHook(() => useChannels(undefined), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("lists connected channels", async () => {
    mockRoute("GET", `/workspaces/${WS}/channels/`, 200, { data: [CHANNEL] });
    const { result } = await renderHook(() => useChannels(WS), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].handle).toBe("ada");
  });
});

describe("useConnectChannel", () => {
  it("requests the authorize URL and sends the app deep link as redirect_uri", async () => {
    mockRoute("POST", `/workspaces/${WS}/channels/connect`, 200, {
      data: { authorize_url: "https://idp.test/auth?state=s" },
    });
    const { result } = await renderHook(() => useConnectChannel(WS), { wrapper: createWrapper() });
    result.current.mutate({ platform: "twitter" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBe("https://idp.test/auth?state=s");
    const call = calls.find((c) => c.url.includes("/channels/connect"));
    expect(call?.body).toMatchObject({ platform: "twitter", redirect_uri: OAUTH_REDIRECT });
  });
});

describe("useCompleteOAuth", () => {
  it("exchanges state+code for the connected channel", async () => {
    mockRoute("GET", "/channels/oauth/callback", 201, { data: CHANNEL });
    const { result } = await renderHook(() => useCompleteOAuth(), { wrapper: createWrapper() });
    result.current.mutate({ state: "s", code: "c" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.handle).toBe("ada");
  });

  it("fails on an invalid state", async () => {
    mockRoute("GET", "/channels/oauth/callback", 400, {
      error: { code: "invalid_state", message: "expired" },
    });
    const { result } = await renderHook(() => useCompleteOAuth(), { wrapper: createWrapper() });
    result.current.mutate({ state: "stale", code: "c" });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("expired");
  });
});

describe("useDisconnectChannel", () => {
  it("disconnects a channel", async () => {
    mockRoute("DELETE", `/workspaces/${WS}/channels/${CHANNEL.id}`, 200, { data: { message: "ok" } });
    const { result } = await renderHook(() => useDisconnectChannel(WS), { wrapper: createWrapper() });
    result.current.mutate({ channelId: CHANNEL.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});
