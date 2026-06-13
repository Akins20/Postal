import { renderHook, waitFor } from "@testing-library/react-native";
import * as SecureStore from "expo-secure-store";

import { useLogin, useLogout, useMe, useSignup } from "@/data/auth";
import { getAccessToken } from "@/lib/secure-session";
import { calls, mockRoute } from "@/test/fetch-mock";
import { createWrapper } from "@/test/react";

const USER = {
  id: "00000000-0000-0000-0000-000000000001",
  email: "ada@example.com",
  email_verified: true,
  status: "active",
  created_at: "2026-01-01T00:00:00Z",
};
const TOKEN = {
  access_token: "acc-1",
  token_type: "Bearer",
  expires_in: 900,
  csrf_token: "c",
  refresh_token: "ref-1",
  user: USER,
};

describe("useMe", () => {
  it("returns null on 401 (signed out is not an error)", async () => {
    mockRoute("GET", "/auth/me", 401, { error: { code: "unauthorized", message: "no" } });
    const { result } = await renderHook(() => useMe(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toBeNull();
  });

  it("returns the user when signed in", async () => {
    mockRoute("GET", "/auth/me", 200, { data: USER });
    const { result } = await renderHook(() => useMe(), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.email).toBe("ada@example.com");
  });
});

describe("useLogin", () => {
  it("stores the session (access in memory, refresh in Keystore) and returns the user", async () => {
    mockRoute("POST", "/auth/login", 200, { data: TOKEN });
    const { result } = await renderHook(() => useLogin(), { wrapper: createWrapper() });
    result.current.mutate({ email: "ada@example.com", password: "correct horse" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.email).toBe("ada@example.com");
    expect(getAccessToken()).toBe("acc-1");
    expect(SecureStore.setItemAsync).toHaveBeenCalledWith("postal.refreshToken", "ref-1");
  });

  it("surfaces a normalized error on bad credentials", async () => {
    mockRoute("POST", "/auth/login", 401, {
      error: { code: "invalid_credentials", message: "Invalid email or password" },
    });
    const { result } = await renderHook(() => useLogin(), { wrapper: createWrapper() });
    result.current.mutate({ email: "ada@example.com", password: "wrong" });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("Invalid email or password");
  });
});

describe("useSignup", () => {
  it("signs up then logs in to obtain a session", async () => {
    mockRoute("POST", "/auth/signup", 201, { data: USER });
    mockRoute("POST", "/auth/login", 200, { data: TOKEN });
    const { result } = await renderHook(() => useSignup(), { wrapper: createWrapper() });
    result.current.mutate({ email: "ada@example.com", password: "correct horse" });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(getAccessToken()).toBe("acc-1");
    expect(calls.some((c) => c.url.includes("/auth/signup"))).toBe(true);
    expect(calls.some((c) => c.url.includes("/auth/login"))).toBe(true);
  });
});

describe("useLogout", () => {
  it("clears the session", async () => {
    mockRoute("POST", "/auth/logout", 200, { data: { message: "ok" } });
    const { result } = await renderHook(() => useLogout(), { wrapper: createWrapper() });
    result.current.mutate();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(getAccessToken()).toBeNull();
    expect(SecureStore.deleteItemAsync).toHaveBeenCalledWith("postal.refreshToken");
  });
});
