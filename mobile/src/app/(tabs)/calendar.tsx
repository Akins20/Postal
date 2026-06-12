import { ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Panel } from "@/ui/panel";
import { StatusPill } from "@/ui/status-pill";

/** Calendar screen - placeholder shell; the real feature lands in 15.4. */
export default function CalendarTabScreen() {
  const { palette } = usePalette();
  const insets = useSafeAreaInsets();

  return (
    <ScrollView
      style={{ backgroundColor: palette.surface }}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + space.lg }]}
    >
      <Text style={[styles.title, { color: palette.fg }]}>Calendar</Text>
      <Panel>
        <View style={styles.row}>
          <Text style={[styles.body, { color: palette.fgMuted }]}>Everything scheduled to publish, plus posting slots.</Text>
          <StatusPill tone="accent">15.4</StatusPill>
        </View>
      </Panel>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.lg },
  title: { fontSize: type.display, fontWeight: "700", letterSpacing: -0.5 },
  body: { fontSize: type.body, flex: 1 },
  row: { flexDirection: "row", alignItems: "center", gap: space.md },
});
