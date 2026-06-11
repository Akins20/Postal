import { headers } from "next/headers";
import type { Metadata, Viewport } from "next";

import { Providers } from "./providers";
import "./globals.css";

export const metadata: Metadata = {
  title: "Postal",
  description: "Free, no-paywall social-media scheduling & publishing.",
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#f5f5f7" },
    { media: "(prefers-color-scheme: dark)", color: "#1e1e20" },
  ],
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  // The proxy sets a per-request CSP nonce; pass it to next-themes so its
  // no-flash inline script is allowed under the strict script-src.
  const nonce = (await headers()).get("x-nonce") ?? undefined;

  return (
    <html lang="en" suppressHydrationWarning className="h-full">
      <body className="min-h-full antialiased">
        <a
          href="#main"
          className="focus:bg-elevated focus:text-fg focus:ring-ring sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-100 focus:rounded-md focus:px-3 focus:py-2 focus:text-sm focus:ring-2"
        >
          Skip to content
        </a>
        <Providers nonce={nonce}>{children}</Providers>
      </body>
    </html>
  );
}
