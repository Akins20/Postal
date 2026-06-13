import type { MetadataRoute } from "next";

const siteURL = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";

// Allow crawling of the public marketing/auth surface; keep crawlers out of the
// authenticated app shell and the API (both redirect or require a session, so
// indexing them yields nothing useful and just wastes crawl budget).
export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: [
        "/api/",
        "/oauth/",
        "/channels",
        "/compose",
        "/calendar",
        "/analytics",
        "/integrations",
        "/wallet",
        "/media",
      ],
    },
    sitemap: `${siteURL}/sitemap.xml`,
    host: siteURL,
  };
}
