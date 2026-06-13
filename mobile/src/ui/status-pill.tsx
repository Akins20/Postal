import { StyleSheet, Text, View } from "react-native";

import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";

export type PillTone = "neutral" | "accent" | "success" | "warning" | "danger";

/**
 * Status badge - text plus a dot, never color-only (same a11y rule as web).
 */
export function StatusPill({ tone = "neutral", children }: { tone?: PillTone; children: string }) {
  const { palette } = usePalette();
  const color =
    tone === "accent"
      ? palette.accent
      : tone === "success"
        ? palette.success
        : tone === "warning"
          ? palette.warning
          : tone === "danger"
            ? palette.danger
            : palette.fgMuted;

  return (
    <View style={[styles.pill, { backgroundColor: `${color}22` }]}>
      <View style={[styles.dot, { backgroundColor: color }]} />
      <Text style={[styles.label, { color }]}>{children}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  pill: {
    flexDirection: "row",
    alignItems: "center",
    gap: space.xs + 2,
    paddingHorizontal: space.sm + 2,
    paddingVertical: 3,
    borderRadius: radius.full,
    alignSelf: "flex-start",
  },
  dot: { width: 6, height: 6, borderRadius: 3 },
  label: { fontSize: type.caption, fontWeight: "600" },
});
