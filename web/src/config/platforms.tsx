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
  /** True when the platform connects with user-supplied credentials (a form),
   * not an OAuth redirect (e.g. Telegram: bot token + chat id). */
  manual?: boolean;
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
    key: "facebook",
    label: "Facebook",
    icon: FacebookGlyph,
    hint: "Publish text, links, photos, and videos to a Facebook Page.",
    caveat: "Posts to a Facebook Page you manage, not a personal profile.",
  },
  {
    key: "telegram",
    label: "Telegram",
    icon: TelegramGlyph,
    hint: "Publish to a Telegram channel or group via your own bot.",
    manual: true,
    caveat: "Create a bot with @BotFather, add it as an admin, then enter its token and chat id.",
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

/** Telegram brand glyph (paper plane), filled like the X mark. */
function TelegramGlyph(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden width="1em" height="1em" {...props}>
      <path d="M21.94 4.66a1 1 0 0 0-1.06-.14L2.6 11.84c-.84.33-.82 1.53.03 1.84l4.34 1.57 1.7 5.16c.2.6.95.78 1.4.33l2.4-2.36 4.5 3.31c.5.37 1.22.1 1.36-.51L23 5.7a1 1 0 0 0-1.06-1.04ZM9.4 14.2l-.27 4.06-1.3-3.95 9.2-6.66-7.63 6.55Z" />
    </svg>
  );
}

/** Facebook brand glyph (the "f" mark), filled like the X mark. */
function FacebookGlyph(props: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden width="1em" height="1em" {...props}>
      <path d="M24 12.07C24 5.4 18.63 0 12 0S0 5.4 0 12.07c0 6.03 4.39 11.03 10.13 11.93v-8.44H7.08v-3.49h3.05V9.41c0-3.02 1.79-4.69 4.53-4.69 1.31 0 2.68.24 2.68.24v2.97h-1.51c-1.49 0-1.96.93-1.96 1.89v2.25h3.33l-.53 3.49h-2.8V24C19.61 23.1 24 18.1 24 12.07z" />
    </svg>
  );
}

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
