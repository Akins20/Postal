import { headers } from "next/headers";
import type { Metadata, Viewport } from "next";
import { Inter } from "next/font/google";

import { Providers } from "./providers";
import "./globals.css";

// Inter (variable) - self-hosted by next/font. The previous system-font stack
// fell back to Arial-likes off macOS, which read as dated everywhere else.
const inter = Inter({
  subsets: ["latin"],
  display: "swap",
  variable: "--font-inter",
});

// Absolute base for OpenGraph/canonical URLs. Production sets the real domain;
// local dev falls back so build-time metadata resolution never throws.
const siteURL = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";

const description =
  "Postal is a free, no-paywall social-media scheduling and publishing platform. " +
  "Compose once, schedule, and publish to X, Instagram, and TikTok from one calendar.";

export const metadata: Metadata = {
  metadataBase: new URL(siteURL),
  title: {
    default: "Postal — Free social-media scheduling & publishing",
    template: "%s · Postal",
  },
  description,
  applicationName: "Postal",
  keywords: [
    "social media scheduling",
    "free Buffer alternative",
    "schedule posts",
    "X scheduling",
    "Instagram scheduling",
    "TikTok scheduling",
    "social media calendar",
    "publish everywhere",
    "no paywall",
  ],
  authors: [{ name: "Postal" }],
  creator: "Postal",
  publisher: "Postal",
  alternates: { canonical: "/" },
  category: "productivity",
  openGraph: {
    type: "website",
    siteName: "Postal",
    title: "Postal — Free social-media scheduling & publishing",
    description,
    url: "/",
    locale: "en_US",
  },
  twitter: {
    card: "summary_large_image",
    title: "Postal — Free social-media scheduling & publishing",
    description,
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-image-preview": "large",
      "max-snippet": -1,
      "max-video-preview": -1,
    },
  },
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
    <html lang="en" suppressHydrationWarning className={`h-full ${inter.variable}`}>
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
