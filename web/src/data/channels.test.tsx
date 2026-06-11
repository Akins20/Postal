import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/react";

import { useChannels, useCompleteOAuth, useConnectChannel, useDisconnectChannel } from "./channels";

const WS_ID = "11111111-1111-1111-1111-111111111111";
const CHANNEL = {
  id: "22222222-2222-2222-2222-222222222222",
  platform: "twitter",
  platform_account_id: "1234567890",
  handle: "ada",
  display_name: "Ada Lovelace",
  status: "active",
  connected_by: "00000000-0000-0000-0000-000000000001",
  created_at: "2026-01-01T00:00:00Z",
};

describe("useChannels", () => {
  it("stays idle without a workspace id", () => {
    const { result } = renderHook(() => useChannels(undefined), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("lists connected channels", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/channels/`, () =>
        HttpResponse.json({ data: [CHANNEL] }),
      ),
    );
    const { result } = renderHook(() => useChannels(WS_ID), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].handle).toBe("ada");
    expect(result.current.data?.[0].status).toBe("active");
  });

  it("returns an empty list when nothing is connected", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/channels/`, () =>
        HttpResponse.json({ data: [] }),
      ),
    );
    const { result } = renderHook(() => useChannels(WS_ID), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([]);
  });
});

describe("useConnectChannel", () => {
  it("returns the authorize URL", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/channels/connect`, () =>
        HttpResponse.json({ data: { authorize_url: "https://x.test/oauth?state=s" } }),
      ),
    );
    const { result } = renderHook(() => useConnectChannel(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ platform: "twitter" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBe("https://x.test/oauth?state=s");
  });

  it("surfaces a normalized error when forbidden", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/channels/connect`, () =>
        HttpResponse.json(
          { error: { code: "forbidden", message: "missing capability" } },
          { status: 403 },
        ),
      ),
    );
    const { result } = renderHook(() => useConnectChannel(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ platform: "twitter" });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.status).toBe(403);
  });
});

describe("useCompleteOAuth", () => {
  it("exchanges state+code for the connected channel", async () => {
    server.use(
      http.get("http://localhost/api/v1/channels/oauth/callback", ({ request }) => {
        const url = new URL(request.url);
        if (url.searchParams.get("state") !== "s1" || url.searchParams.get("code") !== "c1") {
          return HttpResponse.json(
            { error: { code: "validation", message: "bad state" } },
            { status: 400 },
          );
        }
        return HttpResponse.json({ data: CHANNEL }, { status: 201 });
      }),
    );
    const { result } = renderHook(() => useCompleteOAuth(), { wrapper: createWrapper() });
    result.current.mutate({ state: "s1", code: "c1" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.handle).toBe("ada");
  });

  it("fails on an invalid state", async () => {
    server.use(
      http.get("http://localhost/api/v1/channels/oauth/callback", () =>
        HttpResponse.json(
          { error: { code: "validation", message: "state expired" } },
          { status: 400 },
        ),
      ),
    );
    const { result } = renderHook(() => useCompleteOAuth(), { wrapper: createWrapper() });
    result.current.mutate({ state: "stale", code: "c" });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("state expired");
  });
});

describe("useDisconnectChannel", () => {
  it("disconnects a channel", async () => {
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS_ID}/channels/${CHANNEL.id}`, () =>
        HttpResponse.json({ data: { message: "disconnected" } }),
      ),
    );
    const { result } = renderHook(() => useDisconnectChannel(WS_ID), {
      wrapper: createWrapper(),
    });
    result.current.mutate({ channelId: CHANNEL.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  it("surfaces not-found", async () => {
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS_ID}/channels/${CHANNEL.id}`, () =>
        HttpResponse.json(
          { error: { code: "not_found", message: "channel not found" } },
          { status: 404 },
        ),
      ),
    );
    const { result } = renderHook(() => useDisconnectChannel(WS_ID), {
      wrapper: createWrapper(),
    });
    result.current.mutate({ channelId: CHANNEL.id });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.status).toBe(404);
  });
});
