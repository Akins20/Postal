import type { ReactNode } from "react";
import { StyleSheet, View, type ViewStyle } from "react-native";

import { radius, space } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";

/**
 * The Panel card - mobile counterpart of the web's vibrancy Panel: rounded,
 * hairline border, elevated surface. (Blur material arrives with the sheet
 * work; Android renders the opaque fallback by default, same rule as web.)
 */
export function Panel({ children, style }: { children: ReactNode; style?: ViewStyle }) {
  const { palette } = usePalette();
  return (
    <View
      style={[
        styles.panel,
        { backgroundColor: palette.elevated, borderColor: palette.separator },
        style,
      ]}
    >
      {children}
    </View>
  );
}

const styles = StyleSheet.create({
  panel: {
    borderRadius: radius.lg,
    borderWidth: StyleSheet.hairlineWidth,
    padding: space.lg,
    // Soft layered elevation, macOS-flavored.
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.08,
    shadowRadius: 12,
    elevation: 2,
  },
});
