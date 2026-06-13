import { Eye, EyeOff } from "lucide-react-native";
import { useState } from "react";
import {
  Pressable,
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
 * Secure (password) fields render a show/hide toggle.
 */
export function FormField({
  label,
  error,
  hint,
  style,
  secureTextEntry,
  ...input
}: {
  label: string;
  error?: string;
  hint?: string;
} & TextInputProps) {
  const { palette } = usePalette();
  const [focused, setFocused] = useState(false);
  const [revealed, setRevealed] = useState(false);
  const isSecure = !!secureTextEntry;
  const borderColor = error
    ? palette.danger
    : focused
      ? palette.accent
      : palette.separator;

  return (
    <View style={[styles.wrap, style]}>
      <Text style={[styles.label, { color: palette.fg }]}>{label}</Text>
      <View style={styles.inputRow}>
        <TextInput
          accessibilityLabel={label}
          placeholderTextColor={palette.fgSubtle}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          secureTextEntry={isSecure && !revealed}
          style={[
            styles.input,
            isSecure && styles.inputSecure,
            { color: palette.fg, backgroundColor: palette.elevated, borderColor },
          ]}
          {...input}
        />
        {isSecure && (
          <Pressable
            onPress={() => setRevealed((v) => !v)}
            accessibilityRole="button"
            accessibilityLabel={revealed ? "Hide password" : "Show password"}
            hitSlop={8}
            style={styles.toggle}
          >
            {revealed ? (
              <EyeOff size={18} color={palette.fgSubtle} />
            ) : (
              <Eye size={18} color={palette.fgSubtle} />
            )}
          </Pressable>
        )}
      </View>
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
  inputRow: { position: "relative", justifyContent: "center" },
  input: {
    minHeight: 46,
    borderWidth: StyleSheet.hairlineWidth,
    borderRadius: radius.md,
    paddingHorizontal: space.md,
    fontSize: type.subhead,
  },
  inputSecure: { paddingRight: space.xl + space.md },
  toggle: {
    position: "absolute",
    right: 0,
    height: "100%",
    width: space.xl + space.md,
    alignItems: "center",
    justifyContent: "center",
  },
  hint: { fontSize: type.caption },
  error: { fontSize: type.caption, fontWeight: "500" },
});
