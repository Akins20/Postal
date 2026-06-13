import type { ReactNode } from "react";
import {
  KeyboardAvoidingView,
  Platform,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Logo } from "@/ui/logo";

/** Centered logo + title lockup wrapping each auth form (login/signup/reset). */
export function AuthScaffold({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string;
  subtitle?: string;
  children: ReactNode;
  footer?: ReactNode;
}) {
  const { palette } = usePalette();
  const insets = useSafeAreaInsets();
  return (
    <KeyboardAvoidingView
      style={{ flex: 1, backgroundColor: palette.surface }}
      behavior={Platform.OS === "ios" ? "padding" : undefined}
    >
      <ScrollView
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={[
          styles.content,
          { paddingTop: insets.top + space.xxl, paddingBottom: insets.bottom + space.xl },
        ]}
      >
        <View style={styles.brand}>
          <Logo size={64} />
          <Text style={[styles.title, { color: palette.fg }]}>{title}</Text>
          {subtitle && (
            <Text style={[styles.subtitle, { color: palette.fgMuted }]}>{subtitle}</Text>
          )}
        </View>
        <View style={styles.form}>{children}</View>
        {footer && <View style={styles.footer}>{footer}</View>}
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  content: { paddingHorizontal: space.xl, gap: space.xl },
  brand: { alignItems: "center", gap: space.sm },
  title: { fontSize: type.title, fontWeight: "700", letterSpacing: -0.5, marginTop: space.sm },
  subtitle: { fontSize: type.body, textAlign: "center" },
  form: { gap: space.lg },
  footer: { alignItems: "center", gap: space.sm },
});
