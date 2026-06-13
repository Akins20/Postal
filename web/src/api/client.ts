import createClient from "openapi-fetch";

import { logger } from "@/lib/logger";

import type { paths } from "./schema";

/**
 * The single configured API client (FRONTEND_PLAN §7). Typed end-to-end from the
 * frozen OpenAPI contract (`./schema` is generated from docs/openapi.yaml). All
 * requests are same-origin (Next proxy → Go API) with httpOnly session cookies;
 * mutations carry the `X-CSRF-Token` double-submit; a `401` (or a `403` CSRF)
 * triggers a single refresh-and-retry. Session/access tokens are never read in JS.
 */

// The OpenAPI paths already include `/api/v1`, so the client base is the ORIGIN.
// Empty = same-origin (Next proxy → Go API). Tests set an absolute origin (node
// fetch can't resolve relative URLs) via NEXT_PUBLIC_API_BASE.
export const API_ORIGIN = process.env.NEXT_PUBLIC_API_BASE ?? "";

const MUTATING = new Set(["POST", "PUT", "PATCH", "DELETE"]);

/** Read a non-httpOnly cookie value (only `postal_csrf` is JS-readable). */
function readCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
}

// The CSRF token mirrors the `postal_csrf` cookie. We capture it from the auth
// response BODY (login/refresh return it) into sessionStorage, because reading
// the cookie via document.cookie is unreliable behind a cross-subdomain proxy
// (the cookie reaches the API on requests, but JS can't always read it). The
// stored value is the authoritative source; the cookie is only a fallback.
const CSRF_STORE_KEY = "postal_csrf_token";

function storeCsrf(token: string | null): void {
  if (!token || typeof sessionStorage === "undefined") return;
  try {
    sessionStorage.setItem(CSRF_STORE_KEY, token);
  } catch {
    /* storage may be unavailable (private mode); fall back to the cookie */
  }
}

/** CSRF double-submit token: stored value first, cookie as a fallback. */
export function csrfToken(): string | null {
  if (typeof sessionStorage !== "undefined") {
    try {
      const stored = sessionStorage.getItem(CSRF_STORE_KEY);
      if (stored) return stored;
    } catch {
      /* ignore */
    }
  }
  return readCookie("postal_csrf");
}

/** Pull `data.csrf_token` out of an auth response body and persist it. */
async function captureCsrf(res: Response, url: string): Promise<void> {
  if (!res.ok) return;
  if (!url.includes("/auth/login") && !url.includes("/auth/refresh")) return;
  try {
    const body = (await res.clone().json()) as { data?: { csrf_token?: string } };
    if (body?.data?.csrf_token) storeCsrf(body.data.csrf_token);
  } catch {
    /* non-JSON or already-consumed body: ignore */
  }
}

// Single-flight refresh: concurrent 401s share one refresh attempt.
let refreshing: Promise<boolean> | null = null;

function refreshSession(): Promise<boolean> {
  refreshing ??= (async () => {
    try {
      const csrf = csrfToken();
      const res = await fetch(`${API_ORIGIN}/api/v1/auth/refresh`, {
        method: "POST",
        credentials: "include",
        headers: csrf ? { "X-CSRF-Token": csrf } : {},
      });
      // Refresh rotates the CSRF token; capture the fresh one for later mutations.
      await captureCsrf(res, "/auth/refresh");
      return res.ok;
    } catch {
      return false;
    } finally {
      refreshing = null;
    }
  })();
  return refreshing;
}

/** Did this response fail CSRF validation (so a refresh + retry can recover)? */
async function isCsrfFailure(res: Response): Promise<boolean> {
  if (res.status !== 403) return false;
  try {
    const body = (await res.clone().json()) as { error?: { code?: string } };
    return body?.error?.code === "csrf_failed";
  } catch {
    return false;
  }
}

/** fetch wrapper: CSRF on mutations, request-id correlation, refresh-once on 401/CSRF. */
async function apiFetch(request: Request): Promise<Response> {
  const mutating = MUTATING.has(request.method.toUpperCase());
  if (mutating) {
    const csrf = csrfToken();
    if (csrf) request.headers.set("X-CSRF-Token", csrf);
  }
  request.headers.set("X-Request-Id", crypto.randomUUID());

  const retry = request.clone();
  let res = await fetch(request);
  await captureCsrf(res, request.url);

  // Refresh-once on an expired session (401) or a stale CSRF token (403), except
  // the auth endpoints that would loop. The refresh re-seeds the CSRF token.
  const isAuthFlow =
    request.url.includes("/auth/refresh") ||
    request.url.includes("/auth/login") ||
    request.url.includes("/auth/logout");
  const recoverable = res.status === 401 || (mutating && (await isCsrfFailure(res)));
  if (recoverable && !isAuthFlow) {
    if (await refreshSession()) {
      const csrf = csrfToken();
      if (mutating && csrf) retry.headers.set("X-CSRF-Token", csrf);
      res = await fetch(retry);
    }
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
  baseUrl: API_ORIGIN,
  credentials: "include",
  fetch: apiFetch,
});
