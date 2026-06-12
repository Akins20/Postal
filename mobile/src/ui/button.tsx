import type { ReactNode } from "react";
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  type ViewStyle,
} from "react-native";

import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

/**
 * The app button - same variants as the web (primary/secondary/ghost/danger),
 * pressed-state scale, 44pt minimum touch target (WCAG 2.2).
 */
export function Button({
  children,
  onPress,
  variant = "primary",
  disabled = false,
  loading = false,
  style,
}: {
  children: ReactNode;
  onPress?: () => void;
  variant?: ButtonVariant;
  disabled?: boolean;
  loading?: boolean;
  style?: ViewStyle;
}) {
  const { palette } = usePalette();

  const background =
    variant === "primary"
      ? palette.accent
      : variant === "danger"
        ? palette.danger
        : variant === "secondary"
          ? palette.elevated
          : "transparent";
  const textColor =
    variant === "primary" || variant === "danger" ? palette.accentFg : palette.fg;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityState={{ disabled: disabled || loading }}
      disabled={disabled || loading}
      onPress={onPress}
      style={({ pressed }) => [
        styles.base,
        {
          backgroundColor: background,
          borderColor: variant === "secondary" ? palette.separator : "transparent",
          borderWidth: variant === "secondary" ? StyleSheet.hairlineWidth : 0,
          opacity: disabled ? 0.5 : 1,
          transform: [{ scale: pressed ? 0.98 : 1 }],
        },
        style,
      ]}
    >
      {loading ? (
        <ActivityIndicator color={textColor} />
      ) : (
        <Text style={[styles.label, { color: textColor }]}>{children}</Text>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    minHeight: 44,
    paddingHorizontal: space.lg,
    paddingVertical: space.sm,
    borderRadius: radius.md,
    alignItems: "center",
    justifyContent: "center",
    flexDirection: "row",
    gap: space.sm,
  },
  label: {
    fontSize: type.body,
    fontWeight: "600",
  },
});
