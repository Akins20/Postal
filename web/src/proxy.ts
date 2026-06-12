import { NextResponse, type NextRequest } from "next/server";

/**
 * Proxy (Next 16's renamed Middleware) sets a per-request nonce CSP. Next reads
 * the nonce from the request CSP header and applies it to its own scripts, so we
 * ship a strict `script-src 'self' 'nonce-…' 'strict-dynamic'` (FRONTEND_PLAN
 * §9.1). `style-src` keeps `'unsafe-inline'`: React/Radix/Framer Motion set
 * inline `style` attributes that a nonce can't cover, and inline-style XSS is far
 * lower risk than script injection. Dev adds `'unsafe-eval'` (React debug) and
 * websocket `connect-src` for HMR.
 */
export function proxy(request: NextRequest): NextResponse {
  const nonce = btoa(crypto.randomUUID());
  const isDev = process.env.NODE_ENV === "development";

  const csp = [
    `default-src 'self'`,
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${isDev ? " 'unsafe-eval'" : ""}`,
    `style-src 'self' 'unsafe-inline'`,
    `img-src 'self' blob: data:`,
    `font-src 'self'`,
    `connect-src 'self'${isDev ? " ws: wss:" : ""}`,
    `object-src 'none'`,
    `base-uri 'self'`,
    `form-action 'self'`,
    `frame-ancestors 'none'`,
    `frame-src 'none'`,
    `manifest-src 'self'`,
    ...(isDev ? [] : ["upgrade-insecure-requests"]),
  ].join("; ");

  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-nonce", nonce);
  requestHeaders.set("content-security-policy", csp);

  const response = NextResponse.next({ request: { headers: requestHeaders } });
  response.headers.set("content-security-policy", csp);
  return response;
}

export const config = {
  // Run on pages only - skip the API proxy, static assets, and prefetches.
  matcher: [
    {
      source: "/((?!api|_next/static|_next/image|favicon.ico).*)",
      missing: [
        { type: "header", key: "next-router-prefetch" },
        { type: "header", key: "purpose", value: "prefetch" },
      ],
    },
  ],
};
