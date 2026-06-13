import { useState } from "react";
import { ActivityIndicator, Alert, ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { platformInfo, PLATFORMS } from "@/config/platforms";
import { useChannels, useDisconnectChannel, type Channel } from "@/data/channels";
import { useConnectFlow } from "@/features/channels/use-connect-flow";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { Panel } from "@/ui/panel";
import { StatusPill, type PillTone } from "@/ui/status-pill";

const STATUS: Record<Channel["status"], { label: string; tone: PillTone }> = {
  active: { label: "Active", tone: "success" },
  expired: { label: "Expired", tone: "warning" },
  revoked: { label: "Revoked", tone: "danger" },
};

function ConnectedRow({ workspaceId, channel }: { workspaceId: string; channel: Channel }) {
  const { palette } = usePalette();
  const info = platformInfo(channel.platform);
  const disconnect = useDisconnectChannel(workspaceId);
  const st = STATUS[channel.status];

  const confirm = () =>
    Alert.alert(
      `Disconnect @${channel.handle}?`,
      "Scheduled posts to this account will fail until it is reconnected.",
      [
        { text: "Cancel", style: "cancel" },
        {
          text: "Disconnect",
          style: "destructive",
          onPress: () => disconnect.mutate({ channelId: channel.id }),
        },
      ],
    );

  return (
    <View style={[styles.row, { borderBottomColor: palette.separator }]}>
      <View style={[styles.glyphBox, { backgroundColor: palette.surface }]}>
        <info.Glyph size={18} color={palette.fg} />
      </View>
      <View style={{ flex: 1, minWidth: 0 }}>
        <Text style={[styles.name, { color: palette.fg }]} numberOfLines={1}>
          {channel.display_name}
        </Text>
        <Text style={[styles.sub, { color: palette.fgMuted }]} numberOfLines={1}>
          @{channel.handle} · {info.label}
        </Text>
      </View>
      <StatusPill tone={st.tone}>{st.label}</StatusPill>
      <Button variant="ghost" onPress={confirm} loading={disconnect.isPending} style={styles.discBtn}>
        Disconnect
      </Button>
    </View>
  );
}

function ConnectRow({ workspaceId, platformKey }: { workspaceId: string; platformKey: string }) {
  const { palette } = usePalette();
  const info = platformInfo(platformKey);
  const flow = useConnectFlow(workspaceId);
  const [error, setError] = useState<string | null>(null);

  const connect = async () => {
    setError(null);
    const res = await flow.run(platformKey);
    if (res.status === "error") setError(res.message);
  };

  return (
    <View style={[styles.row, { borderBottomColor: palette.separator }]}>
      <View style={[styles.glyphBox, { backgroundColor: palette.surface }]}>
        <info.Glyph size={18} color={palette.fg} />
      </View>
      <View style={{ flex: 1, minWidth: 0 }}>
        <View style={styles.labelRow}>
          <Text style={[styles.name, { color: palette.fg }]}>{info.label}</Text>
          {info.payPerUse && <StatusPill tone="warning">Pay-per-use</StatusPill>}
        </View>
        <Text style={[styles.sub, { color: palette.fgMuted }]}>{info.hint}</Text>
        {info.caveat && <Text style={[styles.caveat, { color: palette.warning }]}>{info.caveat}</Text>}
        {error && (
          <Text accessibilityRole="alert" style={[styles.caveat, { color: palette.danger }]}>
            {error}
          </Text>
        )}
      </View>
      <Button onPress={connect} loading={flow.pending} style={styles.discBtn}>
        Connect
      </Button>
    </View>
  );
}

export default function ChannelsScreen() {
  const { palette } = usePalette();
  const insets = useSafeAreaInsets();
  const { active } = useActiveWorkspace();
  const { data: channels, isPending, isError } = useChannels(active?.id);

  return (
    <ScrollView
      style={{ backgroundColor: palette.surface }}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + space.lg }]}
    >
      <Text style={[styles.title, { color: palette.fg }]}>Channels</Text>

      <Panel>
        <Text style={[styles.cardTitle, { color: palette.fg }]}>Connected accounts</Text>
        {(!active || isPending) && <ActivityIndicator color={palette.fgSubtle} />}
        {isError && (
          <Text accessibilityRole="alert" style={[styles.sub, { color: palette.danger }]}>
            Couldn&apos;t load channels. Pull to retry.
          </Text>
        )}
        {channels?.length === 0 && (
          <Text style={[styles.sub, { color: palette.fgMuted, paddingVertical: space.sm }]}>
            No accounts connected yet. Connect your first below.
          </Text>
        )}
        {active &&
          channels?.map((c) => <ConnectedRow key={c.id} workspaceId={active.id} channel={c} />)}
      </Panel>

      <Panel>
        <Text style={[styles.cardTitle, { color: palette.fg }]}>Connect a platform</Text>
        <Text style={[styles.sub, { color: palette.fgMuted, marginBottom: space.xs }]}>
          You&apos;ll authorize Postal in a secure browser, then land right back here.
        </Text>
        {active &&
          PLATFORMS.map((p) => (
            <ConnectRow key={p.key} workspaceId={active.id} platformKey={p.key} />
          ))}
      </Panel>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.lg, paddingBottom: space.xxl },
  title: { fontSize: type.display, fontWeight: "700", letterSpacing: -0.5 },
  cardTitle: { fontSize: type.body, fontWeight: "600", marginBottom: space.sm },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: space.sm,
    paddingVertical: space.md,
    borderBottomWidth: StyleSheet.hairlineWidth,
  },
  glyphBox: { width: 40, height: 40, borderRadius: 10, alignItems: "center", justifyContent: "center" },
  labelRow: { flexDirection: "row", alignItems: "center", gap: space.sm },
  name: { fontSize: type.body, fontWeight: "600" },
  sub: { fontSize: type.caption + 1 },
  caveat: { fontSize: type.caption, marginTop: 2 },
  discBtn: { minHeight: 36, paddingHorizontal: space.md },
});
