import { useRouter } from "expo-router";
import { CircleCheck, Mail } from "lucide-react-native";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { useLogout, useMe } from "@/data/auth";
import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { Panel } from "@/ui/panel";
import { StatusPill } from "@/ui/status-pill";

/** Home: greeting, account state, and the live overview (widgets land in 15.4). */
export default function HomeScreen() {
  const { palette } = usePalette();
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const { data: user } = useMe();
  const logout = useLogout();

  const signOut = async () => {
    await logout.mutateAsync();
    router.replace("/login");
  };

  return (
    <ScrollView
      style={{ backgroundColor: palette.surface }}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + space.lg }]}
    >
      <View>
        <Text style={[styles.kicker, { color: palette.fgSubtle }]}>Welcome back</Text>
        <Text style={[styles.title, { color: palette.fg }]}>Postal</Text>
      </View>

      <Panel>
        <Text style={[styles.cardTitle, { color: palette.fg }]}>Account</Text>
        <View style={styles.row}>
          <Text style={[styles.body, { color: palette.fgMuted }]} numberOfLines={1}>
            {user?.email ?? ""}
          </Text>
          {user?.email_verified ? (
            <StatusPill tone="success">Verified</StatusPill>
          ) : (
            <StatusPill tone="warning">Unverified</StatusPill>
          )}
        </View>
        {!user?.email_verified && (
          <View style={[styles.note, { borderColor: palette.separator }]}>
            <Mail color={palette.warning} size={16} />
            <Text style={[styles.noteText, { color: palette.fgMuted }]}>
              Check your inbox to verify your email. You can keep using Postal in the meantime.
            </Text>
          </View>
        )}
      </Panel>

      <Panel>
        <View style={styles.row}>
          <CircleCheck color={palette.success} size={18} />
          <Text style={[styles.body, { color: palette.fgMuted, flex: 1 }]}>
            Your scheduled posts, drafts, and channel health will appear here.
          </Text>
        </View>
      </Panel>

      <Button variant="secondary" onPress={signOut} loading={logout.isPending}>
        Sign out
      </Button>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.lg, paddingBottom: space.xxl },
  kicker: { fontSize: type.caption, fontWeight: "600", textTransform: "uppercase", letterSpacing: 0.5 },
  title: { fontSize: type.display, fontWeight: "700", letterSpacing: -0.5 },
  cardTitle: { fontSize: type.body, fontWeight: "600", marginBottom: space.sm },
  row: { flexDirection: "row", alignItems: "center", gap: space.sm },
  body: { fontSize: type.body, flex: 1 },
  note: {
    flexDirection: "row",
    gap: space.sm,
    marginTop: space.md,
    paddingTop: space.md,
    borderTopWidth: StyleSheet.hairlineWidth,
    alignItems: "flex-start",
  },
  noteText: { fontSize: type.caption, flex: 1, lineHeight: 17 },
});
