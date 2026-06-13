import { useRouter } from "expo-router";
import {
  BarChart3,
  ChevronRight,
  Settings as SettingsIcon,
  Users as UsersIcon,
  Wallet as WalletIcon,
} from "lucide-react-native";
import type { ComponentType } from "react";
import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Panel } from "@/ui/panel";

const ITEMS: { href: string; label: string; sub: string; Icon: ComponentType<{ size?: number; color?: string }> }[] = [
  { href: "/more/analytics", label: "Analytics", sub: "How your posts perform", Icon: BarChart3 },
  { href: "/more/members", label: "Members", sub: "Workspace team and roles", Icon: UsersIcon },
  { href: "/more/wallet", label: "Wallet", sub: "Credits for X publishing", Icon: WalletIcon },
  { href: "/more/settings", label: "Settings", sub: "Account and appearance", Icon: SettingsIcon },
];

export default function MoreMenu() {
  const { palette } = usePalette();
  const insets = useSafeAreaInsets();
  const router = useRouter();
  return (
    <ScrollView
      style={{ backgroundColor: palette.surface }}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + space.lg }]}
    >
      <Text style={[styles.title, { color: palette.fg }]}>More</Text>
      <Panel style={{ padding: 0 }}>
        {ITEMS.map((it, i) => (
          <Pressable
            key={it.href}
            accessibilityRole="link"
            onPress={() => router.push(it.href as never)}
            style={[styles.row, { borderTopColor: palette.separator, borderTopWidth: i === 0 ? 0 : StyleSheet.hairlineWidth }]}
          >
            <View style={[styles.iconBox, { backgroundColor: `${palette.accent}14` }]}>
              <it.Icon size={20} color={palette.accent} />
            </View>
            <View style={{ flex: 1 }}>
              <Text style={[styles.label, { color: palette.fg }]}>{it.label}</Text>
              <Text style={[styles.sub, { color: palette.fgMuted }]}>{it.sub}</Text>
            </View>
            <ChevronRight size={18} color={palette.fgSubtle} />
          </Pressable>
        ))}
      </Panel>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.lg, paddingBottom: space.xxl },
  title: { fontSize: type.display, fontWeight: "700", letterSpacing: -0.5 },
  row: { flexDirection: "row", alignItems: "center", gap: space.md, padding: space.lg },
  iconBox: { width: 40, height: 40, borderRadius: 10, alignItems: "center", justifyContent: "center" },
  label: { fontSize: type.body, fontWeight: "600" },
  sub: { fontSize: type.caption + 1 },
});
