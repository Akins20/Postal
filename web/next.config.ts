import type { NextConfig } from "next";

/**
 * Static, non-CSP security headers applied to every response. The CSP (which
 * needs a per-request nonce) is set in `src/middleware.ts`. See FRONTEND_PLAN §9.1.
 */
const securityHeaders = [
  { key: "X-Content-Type-Options", value: "nosniff" },
  { key: "X-Frame-Options", value: "DENY" },
  { key: "Referrer-Policy", value: "no-referrer" },
  { key: "Cross-Origin-Opener-Policy", value: "same-origin" },
  {
    key: "Permissions-Policy",
    value: "camera=(), microphone=(), geolocation=(), browsing-topics=()",
  },
];

/**
 * Proxy `/api/*` to the Go backend so the browser sees one origin and the
 * httpOnly session cookies + CSRF "just work" with no CORS (FRONTEND_PLAN §4).
 * The production API lives on a separate subdomain (api.postal.lettstv.com), so
 * we keep proxying server-side in production too — cookies are scoped to
 * `.postal.lettstv.com`, so they ride along whatever subdomain the web app is
 * served from. Override the target with API_PROXY_TARGET.
 */
const apiTarget =
  process.env.API_PROXY_TARGET ??
  (process.env.NODE_ENV === "production"
    ? "https://api.postal.lettstv.com"
    : "http://localhost:8080");

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // The API's collection routes end in "/" (e.g. /api/v1/workspaces/). Without
  // this, Next 308-normalizes them away before the rewrite — an extra round
  // trip on every collection request.
  skipTrailingSlashRedirect: true,
  // Pin the workspace root to web/ (a stray lockfile sits above the repo).
  turbopack: { root: import.meta.dirname },
  async headers() {
    return [{ source: "/:path*", headers: securityHeaders }];
  },
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${apiTarget}/api/:path*` }];
  },
};

export default nextConfig;
