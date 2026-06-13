import type { MetadataRoute } from "next";

const siteURL = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";

// Only the public, indexable surface. The authenticated app shell is gated by a
// session and excluded from robots, so it has no place in the sitemap.
export default function sitemap(): MetadataRoute.Sitemap {
  const lastModified = new Date();
  return [
    { url: `${siteURL}/`, lastModified, changeFrequency: "weekly", priority: 1 },
    { url: `${siteURL}/about`, lastModified, changeFrequency: "monthly", priority: 0.9 },
    { url: `${siteURL}/signup`, lastModified, changeFrequency: "monthly", priority: 0.9 },
    { url: `${siteURL}/login`, lastModified, changeFrequency: "monthly", priority: 0.8 },
    { url: `${siteURL}/terms`, lastModified, changeFrequency: "yearly", priority: 0.4 },
    { url: `${siteURL}/privacy`, lastModified, changeFrequency: "yearly", priority: 0.4 },
    { url: `${siteURL}/contact`, lastModified, changeFrequency: "yearly", priority: 0.3 },
    { url: `${siteURL}/reset`, lastModified, changeFrequency: "yearly", priority: 0.3 },
  ];
}
