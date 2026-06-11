import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/react";

import { mediaDownloadURL, useDeleteMedia, useMedia, useUploadMedia } from "./media";

const WS_ID = "11111111-1111-1111-1111-111111111111";
const ASSET = {
  id: "55555555-5555-5555-5555-555555555555",
  workspace_id: WS_ID,
  kind: "image",
  mime: "image/png",
  width: 100,
  height: 80,
  duration_ms: 0,
  bytes: 1234,
  status: "uploaded",
  created_at: "2026-01-01T00:00:00Z",
};

describe("useMedia", () => {
  it("stays idle without a workspace id", () => {
    const { result } = renderHook(() => useMedia(undefined), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("lists assets", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/media/`, () =>
        HttpResponse.json({ data: [ASSET] }),
      ),
    );
    const { result } = renderHook(() => useMedia(WS_ID), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].mime).toBe("image/png");
  });
});

describe("mediaDownloadURL", () => {
  it("builds the cookie-authenticated bytes URL", () => {
    expect(mediaDownloadURL(WS_ID, ASSET.id)).toBe(
      `http://localhost/api/v1/workspaces/${WS_ID}/media/${ASSET.id}/download`,
    );
  });
});

describe("useUploadMedia", () => {
  // jsdom's XHR doesn't serialize FormData to multipart, so the handler can't
  // parse the form here; multipart correctness is covered against the real
  // backend (e2e/curl). This verifies the XHR plumbing and envelope handling.
  it("uploads a file and returns the asset", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/media/`, () =>
        HttpResponse.json({ data: ASSET }, { status: 201 }),
      ),
    );
    const { result } = renderHook(() => useUploadMedia(WS_ID), { wrapper: createWrapper() });
    const file = new File(["png-bytes"], "pic.png", { type: "image/png" });
    result.current.mutate({ file });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.id).toBe(ASSET.id);
  });

  it("normalizes an oversize/quota rejection", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/media/`, () =>
        HttpResponse.json(
          { error: { code: "quota_exceeded", message: "storage quota exceeded" } },
          { status: 400 },
        ),
      ),
    );
    const { result } = renderHook(() => useUploadMedia(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ file: new File(["x"], "big.png", { type: "image/png" }) });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("storage quota exceeded");
  });
});

describe("useDeleteMedia", () => {
  it("deletes an asset", async () => {
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS_ID}/media/${ASSET.id}`, () =>
        HttpResponse.json({ data: { message: "deleted" } }),
      ),
    );
    const { result } = renderHook(() => useDeleteMedia(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ mediaId: ASSET.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});
