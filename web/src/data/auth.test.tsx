import { renderHook, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { describe, expect, it } from "vitest";

import { server } from "@/test/msw/server";
import { createWrapper } from "@/test/react";

import { useLogin, useMe } from "./auth";

const USER = {
  id: "00000000-0000-0000-0000-000000000001",
  email: "ada@example.com",
  email_verified: true,
  status: "active",
  created_at: "2026-01-01T00:00:00Z",
};

describe("useMe", () => {
  it("returns null when not signed in (401 + failed refresh)", async () => {
    server.use(
      http.get("http://localhost/api/v1/auth/me", () => new HttpResponse(null, { status: 401 })),
      http.post(
        "http://localhost/api/v1/auth/refresh",
        () => new HttpResponse(null, { status: 401 }),
      ),
    );
    const { result } = renderHook(() => useMe(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBeNull();
  });

  it("returns the user when signed in", async () => {
    server.use(
      http.get("http://localhost/api/v1/auth/me", () => HttpResponse.json({ data: USER })),
    );
    const { result } = renderHook(() => useMe(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.email).toBe("ada@example.com");
  });
});

describe("useLogin", () => {
  it("returns the user on success", async () => {
    server.use(
      http.post("http://localhost/api/v1/auth/login", () =>
        HttpResponse.json({
          data: {
            access_token: "t",
            token_type: "Bearer",
            expires_in: 900,
            csrf_token: "c",
            user: USER,
          },
        }),
      ),
    );
    const { result } = renderHook(() => useLogin(), { wrapper: createWrapper() });
    result.current.mutate({ email: "ada@example.com", password: "correct horse" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.email).toBe("ada@example.com");
  });

  it("throws a normalized error on bad credentials", async () => {
    server.use(
      http.post("http://localhost/api/v1/auth/login", () =>
        HttpResponse.json(
          { error: { code: "invalid_credentials", message: "Invalid email or password" } },
          { status: 401 },
        ),
      ),
    );
    const { result } = renderHook(() => useLogin(), { wrapper: createWrapper() });
    result.current.mutate({ email: "ada@example.com", password: "wrong" });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.code).toBe("invalid_credentials");
  });
});
