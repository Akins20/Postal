import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/react";

import { useAddMember, useMembers, useWorkspaces } from "./workspaces";

const WS = {
  id: "11111111-1111-1111-1111-111111111111",
  name: "Personal",
  owner_user_id: "00000000-0000-0000-0000-000000000001",
  plan: "free",
  created_at: "2026-01-01T00:00:00Z",
};
const MEMBER = {
  workspace_id: WS.id,
  user_id: "00000000-0000-0000-0000-000000000002",
  role: "editor",
  permissions: ["read", "create"],
};

describe("useWorkspaces", () => {
  it("lists the user's workspaces", async () => {
    server.use(
      http.get("http://localhost/api/v1/workspaces/", () => HttpResponse.json({ data: [WS] })),
    );
    const { result } = renderHook(() => useWorkspaces(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].name).toBe("Personal");
  });
});

describe("useMembers", () => {
  it("stays idle without a workspace id", () => {
    const { result } = renderHook(() => useMembers(undefined), { wrapper: createWrapper() });
    expect(result.current.fetchStatus).toBe("idle");
  });

  it("lists members for a workspace", async () => {
    server.use(
      http.get(`http://localhost/api/v1/workspaces/${WS.id}/members`, () =>
        HttpResponse.json({ data: [MEMBER] }),
      ),
    );
    const { result } = renderHook(() => useMembers(WS.id), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.[0].role).toBe("editor");
  });
});

describe("useAddMember", () => {
  it("adds a member and returns it", async () => {
    server.use(
      http.post(`http://localhost/api/v1/workspaces/${WS.id}/members`, () =>
        HttpResponse.json({ data: MEMBER }),
      ),
    );
    const { result } = renderHook(() => useAddMember(WS.id), { wrapper: createWrapper() });
    result.current.mutate({ email: "grace@example.com", role: "editor" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.user_id).toBe(MEMBER.user_id);
  });
});
