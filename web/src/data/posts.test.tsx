import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/react";

import {
  useCreatePost,
  useDeletePost,
  usePost,
  usePosts,
  useUpdatePost,
  useUtmPreview,
  useValidatePost,
} from "./posts";

const WS_ID = "11111111-1111-1111-1111-111111111111";
const CH_ID = "22222222-2222-2222-2222-222222222222";
const POST = {
  id: "33333333-3333-3333-3333-333333333333",
  workspace_id: WS_ID,
  author_user_id: null,
  status: "draft",
  created_at: "2026-01-01T00:00:00Z",
  variants: [{ id: "44444444-4444-4444-4444-444444444444", channel_id: CH_ID, body: "Hello" }],
};

describe("usePosts", () => {
  it("stays idle without a workspace id", () => {
    const { result } = renderHook(() => usePosts(undefined), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("lists posts", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/posts/`, () =>
        HttpResponse.json({ data: [POST] }),
      ),
    );
    const { result } = renderHook(() => usePosts(WS_ID), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].status).toBe("draft");
  });
});

describe("usePost", () => {
  it("fetches one post with its variants", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS_ID}/posts/${POST.id}`, () =>
        HttpResponse.json({ data: POST }),
      ),
    );
    const { result } = renderHook(() => usePost(WS_ID, POST.id), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.variants?.[0].body).toBe("Hello");
  });
});

describe("useCreatePost", () => {
  it("creates a draft from variants", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/`, async ({ request }) => {
        const body = (await request.json()) as { variants: { channel_id: string }[] };
        if (body.variants[0]?.channel_id !== CH_ID) {
          return HttpResponse.json(
            { error: { code: "validation", message: "bad channel" } },
            { status: 400 },
          );
        }
        return HttpResponse.json({ data: POST }, { status: 201 });
      }),
    );
    const { result } = renderHook(() => useCreatePost(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ variants: [{ channel_id: CH_ID, body: "Hello" }] });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.id).toBe(POST.id);
  });

  it("surfaces validation errors", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/`, () =>
        HttpResponse.json(
          { error: { code: "validation", message: "at least one variant required" } },
          { status: 400 },
        ),
      ),
    );
    const { result } = renderHook(() => useCreatePost(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ variants: [] });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("at least one variant required");
  });
});

describe("useUpdatePost", () => {
  it("replaces a post's variants", async () => {
    server.use(
      http.put(`http://localhost/api/v1/workspaces/${WS_ID}/posts/${POST.id}`, () =>
        HttpResponse.json({ data: { ...POST, variants: [{ ...POST.variants[0], body: "New" }] } }),
      ),
    );
    const { result } = renderHook(() => useUpdatePost(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ postId: POST.id, variants: [{ channel_id: CH_ID, body: "New" }] });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.variants?.[0].body).toBe("New");
  });
});

describe("useDeletePost", () => {
  it("deletes a post", async () => {
    server.use(
      http.delete(`http://localhost/api/v1/workspaces/${WS_ID}/posts/${POST.id}`, () =>
        HttpResponse.json({ data: { message: "deleted" } }),
      ),
    );
    const { result } = renderHook(() => useDeletePost(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ postId: POST.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });
});

describe("useValidatePost", () => {
  it("returns per-variant verdicts", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/${POST.id}/validate`, () =>
        HttpResponse.json({
          data: {
            variants: [
              { channel_id: CH_ID, valid: false, code: "too_long", message: "exceeds 280" },
            ],
          },
        }),
      ),
    );
    const { result } = renderHook(() => useValidatePost(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ postId: POST.id });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0]).toMatchObject({ valid: false, code: "too_long" });
  });
});

describe("useUtmPreview", () => {
  it("returns the tagged text", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS_ID}/posts/utm-preview`, () =>
        HttpResponse.json({ data: { text: "see https://a.test/?utm_source=postal" } }),
      ),
    );
    const { result } = renderHook(() => useUtmPreview(WS_ID), { wrapper: createWrapper() });
    result.current.mutate({ text: "see https://a.test/", utm: { utm_source: "postal" } });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toContain("utm_source=postal");
  });
});
