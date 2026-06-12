/**
 * Postal design tokens - the mobile port of web/src/app/globals.css. The web
 * palette is authored in oklch; React Native needs static colors, so each
 * value here is the hex/rgba conversion of the same token. Keep BOTH files in
 * sync when the palette changes (token names match 1:1).
 */

export interface Palette {
  /** Window background. */
  surface: string;
  /** Cards, inputs, menus. */
  elevated: string;
  fg: string;
  fgMuted: string;
  fgSubtle: string;
  accent: string;
  accentSoft: string;
  accentFg: string;
  danger: string;
  success: string;
  warning: string;
  separator: string;
  /** Translucent panel fill (vibrancy scrim over blur). */
  vibrancyPanel: string;
  /** Translucent tab-bar / dock fill. */
  vibrancyDock: string;
}

export const palettes: Record<"light" | "dark", Palette> = {
  light: {
    surface: "#f3f5f9", // oklch(0.972 0.005 255)
    elevated: "#ffffff",
    fg: "#1e2433", // oklch(0.21 0.014 260)
    fgMuted: "#5c6577", // oklch(0.45 0.016 260)
    fgSubtle: "#828b9d", // oklch(0.59 0.016 260)
    accent: "#2f6bef", // oklch(0.57 0.20 257), systemBlue
    accentSoft: "#5e8df5", // oklch(0.66 0.16 257)
    accentFg: "#ffffff",
    danger: "#d93a34", // oklch(0.60 0.21 27)
    success: "#1ba35d", // oklch(0.62 0.17 150)
    warning: "#bb7d18", // oklch(0.60 0.15 70)
    separator: "#e0e3ea", // oklch(0.89 0.006 260)
    vibrancyPanel: "rgba(255,255,255,0.86)",
    vibrancyDock: "rgba(252,253,255,0.78)",
  },
  dark: {
    surface: "#15171f", // oklch(0.17 0.012 260)
    elevated: "#222633", // oklch(0.235 0.013 260)
    fg: "#e8eaf0", // oklch(0.93 0.006 260)
    fgMuted: "#a3a9b9", // oklch(0.71 0.012 260)
    fgSubtle: "#747b8d", // oklch(0.55 0.012 260)
    accent: "#5d87ff", // oklch(0.64 0.19 257)
    accentSoft: "#4a72e8", // oklch(0.56 0.16 257)
    accentFg: "#ffffff",
    danger: "#f05547", // oklch(0.66 0.20 25)
    success: "#36c97c", // oklch(0.72 0.17 150)
    warning: "#e8a23d", // oklch(0.74 0.15 70)
    separator: "rgba(255,255,255,0.13)",
    vibrancyPanel: "rgba(34,38,51,0.85)",
    vibrancyDock: "rgba(34,38,51,0.78)",
  },
};

/** Spacing scale (4px base, same rhythm as the web). */
export const space = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 24,
  xxl: 32,
} as const;

/** Corner radii (macOS-ish continuous corners, FRONTEND_PLAN section 5). */
export const radius = {
  sm: 8,
  md: 12,
  lg: 16,
  xl: 22,
  full: 999,
} as const;

/** Type ramp sizes. */
export const type = {
  caption: 12,
  body: 14,
  subhead: 16,
  title: 20,
  display: 28,
} as const;
