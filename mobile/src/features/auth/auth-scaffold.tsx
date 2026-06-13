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
import { Panel } from "@/ui/panel";

/**
 * Branded auth surface wrapping each form (login/signup/reset): a badge-logo +
 * wordmark + tagline lockup over a Panel card holding the form. Mobile twin of
 * the web's split-screen AuthPanel.
 */
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
          <Logo size={72} />
          <Text style={[styles.wordmark, { color: palette.fg }]}>Postal</Text>
          <Text style={[styles.tagline, { color: palette.fgMuted }]}>
            Schedule and publish everywhere.
          </Text>
        </View>

        <Panel style={styles.card}>
          <View style={styles.head}>
            <Text style={[styles.title, { color: palette.fg }]}>{title}</Text>
            {subtitle && (
              <Text style={[styles.subtitle, { color: palette.fgMuted }]}>{subtitle}</Text>
            )}
          </View>
          <View style={styles.form}>{children}</View>
        </Panel>

        {footer && <View style={styles.footer}>{footer}</View>}
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  content: { paddingHorizontal: space.xl, gap: space.xl },
  brand: { alignItems: "center", gap: space.xs },
  wordmark: { fontSize: type.display, fontWeight: "700", letterSpacing: -0.5, marginTop: space.sm },
  tagline: { fontSize: type.body, textAlign: "center" },
  card: { gap: space.lg },
  head: { gap: space.xs },
  title: { fontSize: type.title, fontWeight: "700", letterSpacing: -0.3 },
  subtitle: { fontSize: type.body },
  form: { gap: space.lg },
  footer: { alignItems: "center", gap: space.sm },
});
