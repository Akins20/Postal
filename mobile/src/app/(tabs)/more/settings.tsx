import { useRouter } from "expo-router";
import { ScrollView, StyleSheet, Text, View } from "react-native";

import { useLogout, useMe } from "@/data/auth";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { useThemeStore, type ThemePreference } from "@/stores/theme";
import { Button } from "@/ui/button";
import { Panel } from "@/ui/panel";
import { StatusPill } from "@/ui/status-pill";

const THEMES: { key: ThemePreference; label: string }[] = [
  { key: "system", label: "System" },
  { key: "light", label: "Light" },
  { key: "dark", label: "Dark" },
];

export default function SettingsScreen() {
  const { palette } = usePalette();
  const router = useRouter();
  const { data: user } = useMe();
  const { active } = useActiveWorkspace();
  const logout = useLogout();
  const preference = useThemeStore((s) => s.preference);
  const setPreference = useThemeStore((s) => s.setPreference);

  const signOut = async () => {
    await logout.mutateAsync();
    router.replace("/login");
  };

  return (
    <ScrollView style={{ backgroundColor: palette.surface }} contentContainerStyle={styles.content}>
      <Panel>
        <Text style={[styles.cardTitle, { color: palette.fg }]}>Account</Text>
        <View style={styles.kv}>
          <Text style={[styles.sub, { color: palette.fgMuted }]}>Email</Text>
          <View style={styles.valueRow}>
            <Text style={[styles.sub, { color: palette.fg }]} numberOfLines={1}>{user?.email}</Text>
            {user?.email_verified ? (
              <StatusPill tone="success">Verified</StatusPill>
            ) : (
              <StatusPill tone="warning">Unverified</StatusPill>
            )}
          </View>
        </View>
        {user && (
          <View style={styles.kv}>
            <Text style={[styles.sub, { color: palette.fgMuted }]}>Member since</Text>
            <Text style={[styles.sub, { color: palette.fg }]}>
              {new Date(user.created_at).toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" })}
            </Text>
          </View>
        )}
      </Panel>

      <Panel>
        <Text style={[styles.cardTitle, { color: palette.fg }]}>Appearance</Text>
        <View style={[styles.segment, { borderColor: palette.separator }]}>
          {THEMES.map((t) => {
            const on = preference === t.key;
            return (
              <Text
                key={t.key}
                accessibilityRole="button"
                onPress={() => setPreference(t.key)}
                style={[
                  styles.segItem,
                  { color: on ? palette.accentFg : palette.fgMuted, backgroundColor: on ? palette.accent : "transparent" },
                ]}
              >
                {t.label}
              </Text>
            );
          })}
        </View>
      </Panel>

      {active && (
        <Panel>
          <Text style={[styles.cardTitle, { color: palette.fg }]}>Workspace</Text>
          <View style={styles.kv}>
            <Text style={[styles.sub, { color: palette.fgMuted }]}>Name</Text>
            <Text style={[styles.sub, { color: palette.fg }]}>{active.name}</Text>
          </View>
          <View style={styles.kv}>
            <Text style={[styles.sub, { color: palette.fgMuted }]}>Plan</Text>
            <StatusPill tone="accent">{active.plan}</StatusPill>
          </View>
        </Panel>
      )}

      <Button variant="secondary" onPress={signOut} loading={logout.isPending}>
        Sign out
      </Button>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.lg },
  cardTitle: { fontSize: type.body, fontWeight: "600", marginBottom: space.sm },
  sub: { fontSize: type.caption + 1 },
  kv: { flexDirection: "row", justifyContent: "space-between", alignItems: "center", paddingVertical: space.xs, gap: space.md },
  valueRow: { flexDirection: "row", alignItems: "center", gap: space.sm, flexShrink: 1 },
  segment: { flexDirection: "row", borderWidth: StyleSheet.hairlineWidth, borderRadius: radius.md, overflow: "hidden" },
  segItem: { flex: 1, textAlign: "center", paddingVertical: space.sm, fontSize: type.caption + 1, fontWeight: "600" },
});
