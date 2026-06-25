import { useState } from "react";
import {
  ActivityIndicator,
  Alert,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { platformInfo, PLATFORMS } from "@/config/platforms";
import {
  useChannels,
  useConnectManual,
  useDisconnectChannel,
  type Channel,
} from "@/data/channels";
import { useConnectFlow } from "@/features/channels/use-connect-flow";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import type { NormalizedError } from "@/lib/api-error";
import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { FormField } from "@/ui/form-field";
import { Panel } from "@/ui/panel";
import { StatusPill, type PillTone } from "@/ui/status-pill";

const STATUS: Record<Channel["status"], { label: string; tone: PillTone }> = {
  active: { label: "Active", tone: "success" },
  expired: { label: "Expired", tone: "warning" },
  revoked: { label: "Revoked", tone: "danger" },
};

function ConnectedRow({
  workspaceId,
  channel,
}: {
  workspaceId: string;
  channel: Channel;
}) {
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
        <Text
          style={[styles.sub, { color: palette.fgMuted }]}
          numberOfLines={1}
        >
          @{channel.handle} · {info.label}
        </Text>
      </View>
      <StatusPill tone={st.tone}>{st.label}</StatusPill>
      <Button
        variant="ghost"
        onPress={confirm}
        loading={disconnect.isPending}
        style={styles.discBtn}
      >
        Disconnect
      </Button>
    </View>
  );
}

function ConnectRow({
  workspaceId,
  platformKey,
  connected,
}: {
  workspaceId: string;
  platformKey: string;
  connected: boolean;
}) {
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
          {connected && <StatusPill tone="success">Connected</StatusPill>}
          {info.payPerUse && (
            <StatusPill tone="warning">Pay-per-use</StatusPill>
          )}
        </View>
        <Text style={[styles.sub, { color: palette.fgMuted }]}>
          {info.hint}
        </Text>
        {info.caveat && (
          <Text style={[styles.caveat, { color: palette.warning }]}>
            {info.caveat}
          </Text>
        )}
        {error && (
          <Text
            accessibilityRole="alert"
            style={[styles.caveat, { color: palette.danger }]}
          >
            {error}
          </Text>
        )}
      </View>
      <Button
        onPress={connect}
        loading={flow.pending}
        variant={connected ? "secondary" : "primary"}
        style={styles.discBtn}
      >
        {connected ? "Add another" : "Connect"}
      </Button>
    </View>
  );
}

// Step-by-step Telegram setup, shown inside the manual connect form.
const TELEGRAM_STEPS = [
  "In Telegram, open a chat with @BotFather and send /newbot. Follow the prompts to name your bot; BotFather replies with a token like 123456:ABC-DEF...",
  "Create (or open) the channel or group you want to post to, then add your new bot as an Administrator with the 'Post Messages' permission.",
  "Get the chat id: for a public channel use @yourchannel; for a private channel/group, forward any message from it to @userinfobot (or @getidsbot) to read the numeric id, e.g. -1001234567890.",
  "Paste the bot token and the chat id below, then tap Connect.",
];

function ManualConnectRow({
  workspaceId,
  platformKey,
}: {
  workspaceId: string;
  platformKey: string;
}) {
  const { palette } = usePalette();
  const info = platformInfo(platformKey);
  const connect = useConnectManual(workspaceId);
  const [open, setOpen] = useState(false);
  const [botToken, setBotToken] = useState("");
  const [chatId, setChatId] = useState("");
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    setError(null);
    try {
      await connect.mutateAsync({
        platform: platformKey,
        credentials: { bot_token: botToken.trim(), chat_id: chatId.trim() },
      });
      setOpen(false);
      setBotToken("");
      setChatId("");
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <View
      style={[
        styles.row,
        { borderBottomColor: palette.separator, alignItems: "flex-start" },
      ]}
    >
      <View style={[styles.glyphBox, { backgroundColor: palette.surface }]}>
        <info.Glyph size={18} color={palette.fg} />
      </View>
      <View style={{ flex: 1, minWidth: 0, gap: space.sm }}>
        <Text style={[styles.name, { color: palette.fg }]}>{info.label}</Text>
        <Text style={[styles.sub, { color: palette.fgMuted }]}>
          {info.hint}
        </Text>
        {!open ? (
          <Button onPress={() => setOpen(true)} style={styles.discBtn}>
            Connect
          </Button>
        ) : (
          <View style={{ gap: space.sm }}>
            <Text style={[styles.name, { color: palette.fg }]}>
              How to set up
            </Text>
            {TELEGRAM_STEPS.map((s, i) => (
              <Text key={i} style={[styles.sub, { color: palette.fgMuted }]}>
                {i + 1}. {s}
              </Text>
            ))}
            <FormField
              label="Bot token"
              value={botToken}
              onChangeText={setBotToken}
              autoCapitalize="none"
              placeholder="123456:ABC-DEF..."
            />
            <FormField
              label="Chat ID or @channel"
              value={chatId}
              onChangeText={setChatId}
              autoCapitalize="none"
              placeholder="@mychannel or -1001234567890"
            />
            {error && (
              <Text
                accessibilityRole="alert"
                style={[styles.caveat, { color: palette.danger }]}
              >
                {error}
              </Text>
            )}
            <View style={{ flexDirection: "row", gap: space.sm }}>
              <Button
                onPress={submit}
                loading={connect.isPending}
                style={styles.discBtn}
              >
                Connect
              </Button>
              <Button
                variant="ghost"
                onPress={() => setOpen(false)}
                style={styles.discBtn}
              >
                Cancel
              </Button>
            </View>
          </View>
        )}
      </View>
    </View>
  );
}

export default function ChannelsScreen() {
  const { palette } = usePalette();
  const insets = useSafeAreaInsets();
  const { active } = useActiveWorkspace();
  const { data: channels, isPending, isError } = useChannels(active?.id);
  const connectedPlatforms = new Set((channels ?? []).map((c) => c.platform));

  return (
    <ScrollView
      style={{ backgroundColor: palette.surface }}
      contentContainerStyle={[
        styles.content,
        { paddingTop: insets.top + space.lg },
      ]}
    >
      <Text style={[styles.title, { color: palette.fg }]}>Channels</Text>

      <Panel>
        <Text style={[styles.cardTitle, { color: palette.fg }]}>
          Connected accounts
        </Text>
        {(!active || isPending) && (
          <ActivityIndicator color={palette.fgSubtle} />
        )}
        {isError && (
          <Text
            accessibilityRole="alert"
            style={[styles.sub, { color: palette.danger }]}
          >
            Couldn&apos;t load channels. Pull to retry.
          </Text>
        )}
        {channels?.length === 0 && (
          <Text
            style={[
              styles.sub,
              { color: palette.fgMuted, paddingVertical: space.sm },
            ]}
          >
            No accounts connected yet. Connect your first below.
          </Text>
        )}
        {active &&
          channels?.map((c) => (
            <ConnectedRow key={c.id} workspaceId={active.id} channel={c} />
          ))}
      </Panel>

      <Panel>
        <Text style={[styles.cardTitle, { color: palette.fg }]}>
          Connect a platform
        </Text>
        <Text
          style={[
            styles.sub,
            { color: palette.fgMuted, marginBottom: space.xs },
          ]}
        >
          You&apos;ll authorize Postal in a secure browser, then land right back
          here.
        </Text>
        {active &&
          PLATFORMS.map((p) =>
            p.manual ? (
              <ManualConnectRow
                key={p.key}
                workspaceId={active.id}
                platformKey={p.key}
              />
            ) : (
              <ConnectRow
                key={p.key}
                workspaceId={active.id}
                platformKey={p.key}
                connected={connectedPlatforms.has(p.key)}
              />
            ),
          )}
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
  glyphBox: {
    width: 40,
    height: 40,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
  },
  labelRow: { flexDirection: "row", alignItems: "center", gap: space.sm },
  name: { fontSize: type.body, fontWeight: "600" },
  sub: { fontSize: type.caption + 1 },
  caveat: { fontSize: type.caption, marginTop: 2 },
  discBtn: { minHeight: 36, paddingHorizontal: space.md },
});
