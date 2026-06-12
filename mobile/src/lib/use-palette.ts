import { useColorScheme } from "react-native";

import { palettes, type Palette } from "@/lib/tokens";
import { useThemeStore } from "@/stores/theme";

/**
 * The active palette: the user's override when set, otherwise the system
 * scheme. Every themed component reads colors through this hook.
 */
export function usePalette(): { palette: Palette; scheme: "light" | "dark" } {
  const system = useColorScheme();
  const preference = useThemeStore((s) => s.preference);
  const scheme =
    preference === "system" ? (system === "dark" ? "dark" : "light") : preference;
  return { palette: palettes[scheme], scheme };
}
