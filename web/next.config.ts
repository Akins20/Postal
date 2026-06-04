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
 * Dev: proxy `/api/*` to the Go backend so the browser sees one origin and the
 * httpOnly session cookies + CSRF "just work" with no CORS (FRONTEND_PLAN §4).
 * In production the web app and API are deployed same-site, so no rewrite.
 */
const apiTarget = process.env.API_PROXY_TARGET ?? "http://localhost:8080";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Pin the workspace root to web/ (a stray lockfile sits above the repo).
  turbopack: { root: import.meta.dirname },
  async headers() {
    return [{ source: "/:path*", headers: securityHeaders }];
  },
  async rewrites() {
    if (process.env.NODE_ENV === "production") return [];
    return [{ source: "/api/:path*", destination: `${apiTarget}/api/:path*` }];
  },
};

export default nextConfig;
