/**
 * Deterministic fetch mock for the data-layer tests. Register canned responses
 * by method + URL substring; the harness records calls and serves JSON. Avoids
 * MSW's native-fetch interception complications under jest-expo while still
 * exercising the real openapi-fetch client and apiFetch interceptor.
 */
interface Route {
  method: string;
  match: string;
  status: number;
  body: unknown;
}

const routes: Route[] = [];
export const calls: { method: string; url: string; body?: unknown }[] = [];

export function mockRoute(method: string, match: string, status: number, body: unknown) {
  routes.push({ method: method.toUpperCase(), match, status, body });
}

export function resetFetchMock() {
  routes.length = 0;
  calls.length = 0;
}

export function installFetchMock() {
  globalThis.fetch = (async (input: Request | string, init?: RequestInit) => {
    const req = input instanceof Request ? input : new Request(input, init);
    const method = req.method.toUpperCase();
    const url = req.url;
    let body: unknown;
    try {
      body = method !== "GET" ? await req.clone().json() : undefined;
    } catch {
      body = undefined;
    }
    calls.push({ method, url, body });
    // Last matching route wins (lets a test override a default).
    const route = [...routes].reverse().find((r) => r.method === method && url.includes(r.match));
    if (!route) {
      return new Response(JSON.stringify({ error: { code: "no_mock", message: url } }), {
        status: 501,
      });
    }
    return new Response(JSON.stringify(route.body), {
      status: route.status,
      headers: { "Content-Type": "application/json" },
    });
  }) as typeof fetch;
}
