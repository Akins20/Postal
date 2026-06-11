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
}

/** Platforms a workspace can connect, in display order. X/Twitter is first. */
export const PLATFORMS: PlatformInfo[] = [
  {
    key: "twitter",
    label: "X (Twitter)",
    icon: XGlyph,
    hint: "Publish posts and threads to an X account.",
    charLimit: 280,
  },
];

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
