import createClient from "openapi-fetch";

import { logger } from "@/lib/logger";
import {
  getAccessToken,
  getRefreshToken,
  saveSession,
  setAccessToken,
} from "@/lib/secure-session";
import type { paths } from "./schema";

/**
 * The single configured API client - the mobile twin of web/src/api/client.ts,
 * built for the BEARER flow the backend ships for native clients (no cookies,
 * no CSRF). The access token rides in the Authorization header; a 401 triggers
 * one single-flight refresh-and-retry using the Keystore refresh token, the
 * same shape as the web's apiFetch.
 */

// Dev default = the host loopback as seen from the Android emulator (10.0.2.2
// is the dev machine running `postal serve`). Real builds set the env var.
export const API_ORIGIN = process.env.EXPO_PUBLIC_API_ORIGIN ?? "http://10.0.2.2:8080";

// Auth endpoints must NOT trigger refresh-on-401 (they would loop).
const AUTH_PATHS = ["/auth/refresh", "/auth/login", "/auth/logout"];

// Single-flight refresh: concurrent 401s share one refresh attempt.
let refreshing: Promise<boolean> | null = null;

/** Exchange the stored refresh token for a new token pair. Returns success. */
export function refreshSession(): Promise<boolean> {
  refreshing ??= (async () => {
    try {
      const refresh = await getRefreshToken();
      if (!refresh) return false;
      const res = await fetch(`${API_ORIGIN}/api/v1/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refresh }),
      });
      if (!res.ok) return false;
      const body = (await res.json()) as {
        data?: { access_token?: string; refresh_token?: string };
      };
      if (!body.data?.access_token) return false;
      await saveSession(body.data.access_token, body.data.refresh_token);
      return true;
    } catch {
      return false;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

/** fetch wrapper: Bearer auth, request-id correlation, refresh-once on 401. */
async function apiFetch(request: Request): Promise<Response> {
  const token = getAccessToken();
  if (token) request.headers.set("Authorization", `Bearer ${token}`);
  const rid = globalThis.crypto?.randomUUID?.() ?? Math.random().toString(36).slice(2);
  request.headers.set("X-Request-Id", rid);

  const retry = request.clone();
  let res = await fetch(request);

  const isAuthFlow = AUTH_PATHS.some((p) => request.url.includes(p));
  if (res.status === 401 && !isAuthFlow) {
    if (await refreshSession()) {
      retry.headers.set("Authorization", `Bearer ${getAccessToken() ?? ""}`);
      res = await fetch(retry);
    }
  }
  if (res.status >= 500) {
    logger.warn("api server error", {
      requestId: res.headers.get("x-request-id") ?? undefined,
      status: res.status,
    });
  }
  return res;
}

export { setAccessToken };
export const api = createClient<paths>({ baseUrl: API_ORIGIN, fetch: apiFetch });
