import createClient from "openapi-fetch";

import { logger } from "@/lib/logger";

import type { paths } from "./schema";

/**
 * The single configured API client (FRONTEND_PLAN §7). Typed end-to-end from the
 * frozen OpenAPI contract (`./schema` is generated from docs/openapi.yaml). All
 * requests are same-origin (dev: Next proxy → Go API) with httpOnly session
 * cookies; mutations carry the `X-CSRF-Token` double-submit; a `401` triggers a
 * single refresh-and-retry. Tokens are never read in JS.
 */

export const API_BASE = "/api/v1";

const MUTATING = new Set(["POST", "PUT", "PATCH", "DELETE"]);

/** Read a non-httpOnly cookie value (only `postal_csrf` is JS-readable). */
function readCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
}

// Single-flight refresh: concurrent 401s share one refresh attempt.
let refreshing: Promise<boolean> | null = null;

function refreshSession(): Promise<boolean> {
  refreshing ??= (async () => {
    try {
      const csrf = readCookie("postal_csrf");
      const res = await fetch(`${API_BASE}/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers: csrf ? { "X-CSRF-Token": csrf } : {},
      });
      return res.ok;
    } catch {
      return false;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

/** fetch wrapper: CSRF on mutations, request-id correlation, refresh-once on 401. */
async function apiFetch(request: Request): Promise<Response> {
  if (MUTATING.has(request.method.toUpperCase())) {
    const csrf = readCookie("postal_csrf");
    if (csrf) request.headers.set("X-CSRF-Token", csrf);
  }
  request.headers.set("X-Request-Id", crypto.randomUUID());

  const retry = request.clone();
  let res = await fetch(request);

  if (res.status === 401 && !request.url.includes("/auth/")) {
    if (await refreshSession()) res = await fetch(retry);
  }
  if (res.status >= 500) {
    logger.warn("api server error", {
      requestId: res.headers.get("x-request-id") ?? undefined,
      status: res.status,
      path: new URL(request.url).pathname,
    });
  }
  return res;
}

export const api = createClient<paths>({
  baseUrl: API_BASE,
  credentials: "include",
  fetch: apiFetch,
});
