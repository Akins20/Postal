import { renderHook, waitFor } from "@testing-library/react-native";

import {
  firstURL,
  useCreatePost,
  useDeletePost,
  usePosts,
  useValidatePost,
} from "@/data/posts";
import { calls, mockRoute } from "@/test/fetch-mock";
import { createWrapper } from "@/test/react";

const WS = "11111111-1111-1111-1111-111111111111";
const CH = "22222222-2222-2222-2222-222222222222";
const POST = {
  id: "33333333-3333-3333-3333-333333333333",
  workspace_id: WS,
  status: "draft",
  created_at: "2026-01-01T00:00:00Z",
};

describe("firstURL", () => {
  it("finds the first http(s) link", () => {
    expect(firstURL("see https://a.test/x and http://b.test")).toBe("https://a.test/x");
    expect(firstURL("no links")).toBeUndefined();
  });
});

describe("usePosts", () => {
  it("lists posts", async () => {
    mockRoute("GET", `/workspaces/${WS}/posts/`, 200, { data: [POST] });
    const { result } = await renderHook(() => usePosts(WS), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].status).toBe("draft");
  });
});

describe("useCreatePost", () => {
  it("creates a draft from variants", async () => {
    mockRoute("POST", `/workspaces/${WS}/posts/`, 201, { data: POST });
    const { result } = await renderHook(() => useCreatePost(WS), { wrapper: createWrapper() });
    result.current.mutate({ variants: [{ channel_id: CH, body: "hi" }] });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.id).toBe(POST.id);
    const call = calls.find((c) => c.method === "POST" && c.url.includes("/posts/"));
    expect(call?.body).toMatchObject({ variants: [{ channel_id: CH, body: "hi" }] });
  });

  it("surfaces validation errors", async () => {
    mockRoute("POST", `/workspaces/${WS}/posts/`, 400, {
      error: { code: "validation", message: "at least one variant required" },
    });
    const { result } = await renderHook(() => useCreatePost(WS), { wrapper: createWrapper() });
    result.current.mutate({ variants: [] });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("at least one variant required");
  });
});

describe("useValidatePost", () => {
  it("returns per-variant verdicts", async () => {
    mockRoute("POST", `/workspaces/${WS}/posts/${POST.id}/validate`, 200, {
      data: { variants: [{ channel_id: CH, valid: false, code: "too_long", message: "exceeds 280" }] },
    });
    const { result } = await renderHook(() => useValidatePost(WS), { wrapper: createWrapper() });
    result.current.mutate({ postId: POST.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0]).toMatchObject({ valid: false, code: "too_long" });
  });
});

describe("useDeletePost", () => {
  it("deletes a post", async () => {
    mockRoute("DELETE", `/workspaces/${WS}/posts/${POST.id}`, 200, { data: { message: "ok" } });
    const { result } = await renderHook(() => useDeletePost(WS), { wrapper: createWrapper() });
    result.current.mutate({ postId: POST.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});
