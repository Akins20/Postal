import type { ComponentType } from "react";
import Svg, { Circle, Path, Rect, type SvgProps } from "react-native-svg";

/*
 * Platform registry - the mobile twin of web/src/config/platforms.tsx. Brand
 * glyphs are react-native-svg (lucide v1 dropped brand icons). Each platform
 * declares its publishing constraints so the composer/channels screens can
 * explain caveats (media-required, pay-per-use, business-account, audit).
 */

type GlyphProps = SvgProps & { size?: number; color?: string };

function XGlyph({ size = 20, color = "currentColor", ...p }: GlyphProps) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill={color} {...p}>
      <Path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231 5.45-6.231Zm-1.161 17.52h1.833L7.084 4.126H5.117l11.966 15.644Z" />
    </Svg>
  );
}

function InstagramGlyph({
  size = 20,
  color = "currentColor",
  ...p
}: GlyphProps) {
  return (
    <Svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth={2}
      {...p}
    >
      <Rect x={2} y={2} width={20} height={20} rx={5} />
      <Circle cx={12} cy={12} r={4.2} />
      <Circle cx={17.4} cy={6.6} r={1.1} fill={color} stroke="none" />
    </Svg>
  );
}

function TikTokGlyph({ size = 20, color = "currentColor", ...p }: GlyphProps) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill={color} {...p}>
      <Path d="M19.59 6.69a4.83 4.83 0 0 1-3.77-4.25V2h-3.45v13.67a2.89 2.89 0 0 1-5.2 1.74 2.89 2.89 0 0 1 2.31-4.64 2.93 2.93 0 0 1 .88.13V9.4a6.84 6.84 0 0 0-1-.05A6.33 6.33 0 0 0 5 20.1a6.34 6.34 0 0 0 10.86-4.43v-7a8.16 8.16 0 0 0 4.77 1.52v-3.4a4.85 4.85 0 0 1-1-.1z" />
    </Svg>
  );
}

function FacebookGlyph({
  size = 20,
  color = "currentColor",
  ...p
}: GlyphProps) {
  return (
    <Svg width={size} height={size} viewBox="0 0 24 24" fill={color} {...p}>
      <Path d="M24 12.07C24 5.4 18.63 0 12 0S0 5.4 0 12.07c0 6.03 4.39 11.03 10.13 11.93v-8.44H7.08v-3.49h3.05V9.41c0-3.02 1.79-4.69 4.53-4.69 1.31 0 2.68.24 2.68.24v2.97h-1.51c-1.49 0-1.96.93-1.96 1.89v2.25h3.33l-.53 3.49h-2.8V24C19.61 23.1 24 18.1 24 12.07z" />
    </Svg>
  );
}

export interface PlatformInfo {
  key: string;
  label: string;
  Glyph: ComponentType<GlyphProps>;
  hint: string;
  charLimit?: number;
  /** Publishing here spends wallet credits. */
  payPerUse?: boolean;
  /** Rejects text-only posts (needs image/video). */
  requiresMedia?: boolean;
  /** One-line caveat shown on the connect row. */
  caveat?: string;
}

export const PLATFORMS: PlatformInfo[] = [
  {
    key: "twitter",
    label: "X (Twitter)",
    Glyph: XGlyph,
    hint: "Publish posts and threads to an X account.",
    charLimit: 280,
    payPerUse: true,
  },
  {
    key: "instagram",
    label: "Instagram",
    Glyph: InstagramGlyph,
    hint: "Publish images and Reels to an Instagram Business or Creator account.",
    charLimit: 2200,
    requiresMedia: true,
    caveat:
      "Needs a Business/Creator account linked to a Facebook Page. No text-only posts.",
  },
  {
    key: "facebook",
    label: "Facebook",
    Glyph: FacebookGlyph,
    hint: "Publish text, links, photos, and videos to a Facebook Page.",
    caveat: "Posts to a Facebook Page you manage, not a personal profile.",
  },
  {
    key: "tiktok",
    label: "TikTok",
    Glyph: TikTokGlyph,
    hint: "Publish videos and photo posts to a TikTok account.",
    charLimit: 2200,
    requiresMedia: true,
    caveat:
      "No text-only posts. Until the app passes TikTok's audit, API posts stay private to you.",
  },
];

export function platformInfo(key: string): PlatformInfo {
  return (
    PLATFORMS.find((p) => p.key === key) ?? {
      key,
      label: key,
      Glyph: XGlyph,
      hint: "",
    }
  );
}
