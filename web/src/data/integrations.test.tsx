import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/react";

import { useConfigureIntegration, useIntegrations, useShortenLinks } from "./integrations";

const WS_ID = "11111111-1111-1111-1111-111111111111";
const OG = {
  provider: "ogshortener",
  enabled: false,
  auto_apply: false,
  configured: false,
  updated_at: "2026-06-12T00:00:00Z",
};

describe("useIntegrations", () => {
  it("lists offered providers even before configuration", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/integrations/`, () =>
        HttpResponse.json({ data: [OG] }),
      ),
    );
    const { result } = renderHook(() => useIntegrations(WS_ID), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].provider).toBe("ogshortener");
    expect(result.current.data?.[0].configured).toBe(false);
  });
});

describe("useConfigureIntegration", () => {
  it("submits the key and returns the configured state", async () => {
    let sent: Record<string, unknown> | null = null;
    server.use(
      http.put(
        `http://localhost/api/v1/workspaces/${WS_ID}/integrations/ogshortener`,
        async ({ request }) => {
          sent = (await request.json()) as typeof sent;
          return HttpResponse.json({ data: { ...OG, enabled: true, configured: true } });
        },
      ),
    );
    const { result } = renderHook(() => useConfigureIntegration(WS_ID), {
      wrapper: createWrapper(),
    });
    result.current.mutate({ provider: "ogshortener", enabled: true, apiKey: "ogl_k" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.configured).toBe(true);
    await waitFor(() =>
      expect(sent).toMatchObject({ enabled: true, api_key: "ogl_k", auto_apply: false }),
    );
  });

  it("surfaces a rejected key", async () => {
    server.use(
      http.put(`http://localhost/api/v1/workspaces/${WS_ID}/integrations/ogshortener`, () =>
        HttpResponse.json(
          { error: { code: "invalid_api_key", message: "the provider rejected that API key" } },
          { status: 400 },
        ),
      ),
    );
    const { result } = renderHook(() => useConfigureIntegration(WS_ID), {
      wrapper: createWrapper(),
    });
    result.current.mutate({ provider: "ogshortener", enabled: true, apiKey: "bad" });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.code).toBe("invalid_api_key");
  });
});

describe("useShortenLinks", () => {
  it("returns the rewritten text", async () => {
    server.use(
      http.post(
        `http://localhost/api/v1/workspaces/${WS_ID}/integrations/ogshortener/shorten`,
        () => HttpResponse.json({ data: { text: "see https://ogsh.rt/abc1" } }),
      ),
    );
    const { result } = renderHook(() => useShortenLinks(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ text: "see https://example.com/long" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBe("see https://ogsh.rt/abc1");
  });

  it("surfaces the not-configured error", async () => {
    server.use(
      http.post(
        `http://localhost/api/v1/workspaces/${WS_ID}/integrations/ogshortener/shorten`,
        () =>
          HttpResponse.json(
            {
              error: {
                code: "integration_not_configured",
                message: "enable OGShortener with your API key on the Integrations page first",
              },
            },
            { status: 400 },
          ),
      ),
    );
    const { result } = renderHook(() => useShortenLinks(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ text: "see https://example.com/long" });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.code).toBe("integration_not_configured");
  });
});
