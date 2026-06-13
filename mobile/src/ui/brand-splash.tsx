import { ActivityIndicator, StyleSheet, Text, View } from "react-native";

import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Logo } from "@/ui/logo";

/** Full-screen branded loader shown during session bootstrap / route guards. */
export function BrandSplash() {
  const { palette } = usePalette();
  return (
    <View style={[styles.root, { backgroundColor: palette.surface }]}>
      <Logo size={88} />
      <Text style={[styles.word, { color: palette.fg }]}>Postal</Text>
      <ActivityIndicator color={palette.fgSubtle} style={{ marginTop: space.lg }} />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, alignItems: "center", justifyContent: "center", gap: space.md },
  word: { fontSize: type.title, fontWeight: "700", letterSpacing: -0.5 },
});
