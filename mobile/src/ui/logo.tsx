import Svg, { Defs, LinearGradient, Path, Rect, Stop } from "react-native-svg";

import { usePalette } from "@/lib/use-palette";

/*
 * The Postal mark: a paper plane (publish/send) - the vector twin of the app
 * icon, so the splash, login, and icon all share one shape. `badge` draws the
 * gradient squircle + white plane (app-icon lockup); `glyph` draws just the
 * plane in the accent color for inline use.
 *
 * Plane geometry (0..100 viewBox), pointing up-right:
 *   faceA (top wing):   nose -> back-left -> fold
 *   faceB (lower wing):  nose -> fold -> tail
 */
const FACE_A = "M93 8 L7 46 L40 60 Z";
const FACE_B = "M93 8 L40 60 L52 93 Z";

export function Logo({
  size = 64,
  variant = "badge",
}: {
  size?: number;
  variant?: "badge" | "glyph";
}) {
  const { palette } = usePalette();

  if (variant === "glyph") {
    return (
      <Svg width={size} height={size} viewBox="0 0 100 100" accessibilityLabel="Postal">
        <Path d={FACE_B} fill={palette.accentSoft} />
        <Path d={FACE_A} fill={palette.accent} />
      </Svg>
    );
  }

  const r = size * 0.225;
  return (
    <Svg width={size} height={size} viewBox="0 0 100 100" accessibilityLabel="Postal">
      <Defs>
        <LinearGradient id="postalBg" x1="0" y1="0" x2="0" y2="1">
          <Stop offset="0" stopColor={palette.accentSoft} />
          <Stop offset="1" stopColor={palette.accent} />
        </LinearGradient>
      </Defs>
      <Rect x="0" y="0" width="100" height="100" rx={(r / size) * 100} fill="url(#postalBg)" />
      {/* Plane inset within the badge. */}
      <Path d="M76.4 26 L23.6 49.4 L43 57.8 Z" fill="#ffffff" fillOpacity={0.82} />
      <Path d="M76.4 26 L43 57.8 L50.4 78.2 Z" fill="#ffffff" />
    </Svg>
  );
}
