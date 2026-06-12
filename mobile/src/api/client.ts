import createClient from "openapi-fetch";

import type { paths } from "./schema";

/**
 * The single configured API client - the mobile twin of web/src/api/client.ts,
 * built for the BEARER flow the backend ships for native clients: no cookies,
 * no CSRF. The access token lives in memory only; 15.1 adds the Keystore-
 * backed refresh-token store and the single-flight refresh-on-401 interceptor
 * (same shape as the web's apiFetch).
 */

// Dev default targets the host loopback as seen from the Android emulator
// (10.0.2.2 = the machine running `postal serve`). Real devices/builds set
// EXPO_PUBLIC_API_ORIGIN.
export const API_ORIGIN =
  process.env.EXPO_PUBLIC_API_ORIGIN ?? "http://10.0.2.2:8080";

// In-memory access token; set by the session layer (15.1), never persisted.
let accessToken: string | null = null;

/** Install the current access token (null clears it on logout). */
export function setAccessToken(token: string | null) {
  accessToken = token;
}

/** apiFetch decorates every request with auth + request-id correlation. */
async function apiFetch(request: Request): Promise<Response> {
  if (accessToken) {
    request.headers.set("Authorization", `Bearer ${accessToken}`);
  }
  // Hermes ships crypto.randomUUID on current RN; fall back just in case.
  const rid =
    globalThis.crypto?.randomUUID?.() ?? Math.random().toString(36).slice(2);
  request.headers.set("X-Request-Id", rid);
  return fetch(request);
}

export const api = createClient<paths>({ baseUrl: API_ORIGIN, fetch: apiFetch });
