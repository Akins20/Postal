import type { MetadataRoute } from "next";

// PWA / install manifest. Icons reuse the app-router convention assets so the
// brand mark stays in one place. theme_color matches the accent used in the
// dock and the OpenGraph card.
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "Postal — Free social-media scheduling & publishing",
    short_name: "Postal",
    description:
      "Free, no-paywall social-media scheduling and publishing. Compose once, schedule, and publish to X, Instagram, and TikTok.",
    start_url: "/",
    display: "standalone",
    background_color: "#f5f5f7",
    theme_color: "#2f6bef",
    icons: [
      { src: "/favicon.ico", sizes: "any", type: "image/x-icon" },
      { src: "/icon.png", sizes: "512x512", type: "image/png" },
      { src: "/apple-icon.png", sizes: "180x180", type: "image/png" },
    ],
  };
}
