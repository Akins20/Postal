import { useState } from "react";
import {
  StyleSheet,
  Text,
  TextInput,
  View,
  type TextInputProps,
} from "react-native";

import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";

/**
 * Accessible labelled text field: label, input, and an error or hint line.
 * The error gets an alert role and red border; mirrors the web FormField.
 */
export function FormField({
  label,
  error,
  hint,
  style,
  ...input
}: {
  label: string;
  error?: string;
  hint?: string;
} & TextInputProps) {
  const { palette } = usePalette();
  const [focused, setFocused] = useState(false);
  const borderColor = error
    ? palette.danger
    : focused
      ? palette.accent
      : palette.separator;

  return (
    <View style={[styles.wrap, style]}>
      <Text style={[styles.label, { color: palette.fg }]}>{label}</Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={palette.fgSubtle}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        style={[
          styles.input,
          { color: palette.fg, backgroundColor: palette.elevated, borderColor },
        ]}
        {...input}
      />
      {hint && !error && <Text style={[styles.hint, { color: palette.fgMuted }]}>{hint}</Text>}
      {error && (
        <Text accessibilityRole="alert" style={[styles.error, { color: palette.danger }]}>
          {error}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: space.xs + 2 },
  label: { fontSize: type.body, fontWeight: "600" },
  input: {
    minHeight: 46,
    borderWidth: StyleSheet.hairlineWidth,
    borderRadius: radius.md,
    paddingHorizontal: space.md,
    fontSize: type.subhead,
  },
  hint: { fontSize: type.caption },
  error: { fontSize: type.caption, fontWeight: "500" },
});
