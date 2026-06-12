import { Radio } from "lucide-react";
import type { ComponentType, SVGProps } from "react";

/**
 * X (Twitter) brand glyph. Lucide dropped brand icons in v1, so platforms ship
 * their own minimal SVG marks, sized/stroked to sit alongside lucide icons.
 */
function XGlyph(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden width="1em" height="1em" {...props}>
      <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231 5.45-6.231Zm-1.161 17.52h1.833L7.084 4.126H5.117l11.966 15.644Z" />
    </svg>
  );
}

export interface PlatformInfo {
  /** Backend platform key (what POST /channels/connect expects). */
  key: string;
  label: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  /** One-line hint shown next to the connect action. */
  hint: string;
  /** Client-side character cap for the compose counter (server re-validates). */
  charLimit?: number;
  /** True when publishing to this platform spends wallet credits. */
  payPerUse?: boolean;
  /** True when the platform rejects text-only posts (needs image/video). */
  requiresMedia?: boolean;
  /** Extra one-line caveat surfaced on the connect row. */
  caveat?: string;
}

/** Platforms a workspace can connect, in display order. X/Twitter is first. */
export const PLATFORMS: PlatformInfo[] = [
  {
    key: "twitter",
    label: "X (Twitter)",
    icon: XGlyph,
    hint: "Publish posts and threads to an X account.",
    charLimit: 280,
    payPerUse: true,
  },
  {
    key: "instagram",
    label: "Instagram",
    icon: InstagramGlyph,
    hint: "Publish images and Reels to an Instagram Business or Creator account.",
    charLimit: 2200,
    requiresMedia: true,
    caveat: "Needs a Business/Creator account linked to a Facebook Page. No text-only posts.",
  },
  {
    key: "tiktok",
    label: "TikTok",
    icon: TikTokGlyph,
    hint: "Publish videos and photo posts to a TikTok account.",
    charLimit: 2200,
    requiresMedia: true,
    caveat:
      "No text-only posts. Until the app passes TikTok's audit, API posts stay private to you.",
  },
];

/**
 * Instagram brand glyph (camera outline), drawn to sit alongside lucide icons.
 */
function InstagramGlyph(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
      width="1em"
      height="1em"
      {...props}
    >
      <rect x="2" y="2" width="20" height="20" rx="5" />
      <circle cx="12" cy="12" r="4.2" />
      <circle cx="17.4" cy="6.6" r="1.1" fill="currentColor" stroke="none" />
    </svg>
  );
}

/** TikTok brand glyph (musical note), filled like the X mark. */
function TikTokGlyph(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden width="1em" height="1em" {...props}>
      <path d="M19.59 6.69a4.83 4.83 0 0 1-3.77-4.25V2h-3.45v13.67a2.89 2.89 0 0 1-5.2 1.74 2.89 2.89 0 0 1 2.31-4.64 2.93 2.93 0 0 1 .88.13V9.4a6.84 6.84 0 0 0-1-.05A6.33 6.33 0 0 0 5 20.1a6.34 6.34 0 0 0 10.86-4.43v-7a8.16 8.16 0 0 0 4.77 1.52v-3.4a4.85 4.85 0 0 1-1-.1z" />
    </svg>
  );
}

export function platformInfo(key: string): PlatformInfo {
  return (
    PLATFORMS.find((p) => p.key === key) ?? {
      key,
      label: key,
      icon: Radio,
      hint: "",
    }
  );
}
