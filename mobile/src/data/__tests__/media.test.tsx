import { renderHook, waitFor } from "@testing-library/react-native";

import { uploadMedia, useDeleteMedia, useMedia } from "@/data/media";
import { calls, mockRoute } from "@/test/fetch-mock";
import { createWrapper } from "@/test/react";

const WS = "11111111-1111-1111-1111-111111111111";
const ASSET = {
  id: "55555555-5555-5555-5555-555555555555",
  workspace_id: WS,
  kind: "image",
  mime: "image/png",
  width: 10,
  height: 10,
  duration_ms: 0,
  bytes: 2048,
  status: "uploaded",
  created_at: "2026-01-01T00:00:00Z",
};

describe("useMedia", () => {
  it("lists assets", async () => {
    mockRoute("GET", `/workspaces/${WS}/media/`, 200, { data: [ASSET] });
    const { result } = await renderHook(() => useMedia(WS), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].mime).toBe("image/png");
  });
});

describe("uploadMedia", () => {
  it("posts multipart to the media endpoint and returns the asset", async () => {
    mockRoute("POST", `/workspaces/${WS}/media/`, 201, { data: ASSET });
    const asset = await uploadMedia(WS, { uri: "file:///pic.png", name: "pic.png", mime: "image/png" });
    expect(asset.id).toBe(ASSET.id);
    const call = calls.find((c) => c.method === "POST" && c.url.includes("/media/"));
    expect(call).toBeDefined();
  });

  it("normalizes a quota/oversize rejection", async () => {
    mockRoute("POST", `/workspaces/${WS}/media/`, 400, {
      error: { code: "quota_exceeded", message: "storage quota exceeded" },
    });
    await expect(
      uploadMedia(WS, { uri: "file:///big.png", name: "big.png", mime: "image/png" }),
    ).rejects.toMatchObject({ message: "storage quota exceeded" });
  });
});

describe("useDeleteMedia", () => {
  it("deletes an asset", async () => {
    mockRoute("DELETE", `/workspaces/${WS}/media/${ASSET.id}`, 200, { data: { message: "ok" } });
    const { result } = await renderHook(() => useDeleteMedia(WS), { wrapper: createWrapper() });
    result.current.mutate({ mediaId: ASSET.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});
