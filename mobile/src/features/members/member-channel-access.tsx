import { Check } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, Pressable, StyleSheet, Switch, Text, View } from "react-native";

import { useChannels } from "@/data/channels";
import { useMemberChannels, useSetMemberChannels, type ChannelAccess } from "@/data/governance";
import type { NormalizedError } from "@/lib/api-error";
import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";

/** Per-member channel-access disclosure (mirrors the web). */
export function MemberChannelAccess({ workspaceId, userId }: { workspaceId: string; userId: string }) {
  const { palette } = usePalette();
  const [open, setOpen] = useState(false);
  return (
    <View style={styles.wrap}>
      <Pressable onPress={() => setOpen((o) => !o)}>
        <Text style={[styles.toggle, { color: palette.accent }]}>
          {open ? "Hide channel access" : "Channel access"}
        </Text>
      </Pressable>
      {open && <AccessLoader workspaceId={workspaceId} userId={userId} />}
    </View>
  );
}

function AccessLoader({ workspaceId, userId }: { workspaceId: string; userId: string }) {
  const { palette } = usePalette();
  const { data: access, isPending } = useMemberChannels(workspaceId, userId);
  if (isPending || !access) return <ActivityIndicator color={palette.fgSubtle} style={{ marginTop: space.sm }} />;
  return <Editor workspaceId={workspaceId} userId={userId} initial={access} />;
}

function Editor({
  workspaceId,
  userId,
  initial,
}: {
  workspaceId: string;
  userId: string;
  initial: ChannelAccess;
}) {
  const { palette } = usePalette();
  const { data: channels = [] } = useChannels(workspaceId);
  const save = useSetMemberChannels(workspaceId, userId);
  const [restricted, setRestricted] = useState(initial.restricted);
  const [selected, setSelected] = useState<Set<string>>(new Set(initial.allowed_channel_ids));
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const toggle = (id: string) => {
    setSaved(false);
    setSelected((cur) => {
      const next = new Set(cur);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const onSave = async () => {
    setError(null);
    setSaved(false);
    try {
      await save.mutateAsync({ restricted, channel_ids: restricted ? [...selected] : [] });
      setSaved(true);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <View style={[styles.editor, { borderColor: palette.separator }]}>
      <View style={styles.restrictRow}>
        <Text style={[styles.restrictLabel, { color: palette.fg }]}>Restrict to selected</Text>
        <Switch
          value={restricted}
          onValueChange={(v) => {
            setRestricted(v);
            setSaved(false);
          }}
        />
      </View>
      {!restricted ? (
        <Text style={[styles.hint, { color: palette.fgSubtle }]}>Can publish to every channel.</Text>
      ) : channels.length === 0 ? (
        <Text style={[styles.hint, { color: palette.fgSubtle }]}>No channels connected yet.</Text>
      ) : (
        channels.map((c) => {
          const on = selected.has(c.id);
          return (
            <Pressable key={c.id} onPress={() => toggle(c.id)} style={styles.channelRow}>
              <View
                style={[
                  styles.box,
                  { borderColor: on ? palette.accent : palette.separator, backgroundColor: on ? palette.accent : "transparent" },
                ]}
              >
                {on && <Check size={12} color={palette.accentFg} />}
              </View>
              <Text style={[styles.channelName, { color: palette.fg }]}>@{c.handle}</Text>
              <Text style={[styles.hint, { color: palette.fgSubtle }]}>{c.platform}</Text>
            </Pressable>
          );
        })
      )}
      <View style={styles.saveRow}>
        <Button onPress={onSave} loading={save.isPending} style={styles.saveBtn}>
          Save access
        </Button>
        {saved && <Text style={[styles.hint, { color: palette.success }]}>Saved.</Text>}
        {error && <Text style={[styles.hint, { color: palette.danger }]}>{error}</Text>}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { marginTop: space.xs },
  toggle: { fontSize: type.caption, fontWeight: "600" },
  editor: { marginTop: space.sm, borderWidth: StyleSheet.hairlineWidth, borderRadius: radius.md, padding: space.sm, gap: space.sm },
  restrictRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  restrictLabel: { fontSize: type.body, fontWeight: "500" },
  hint: { fontSize: type.caption },
  channelRow: { flexDirection: "row", alignItems: "center", gap: space.sm },
  box: { width: 18, height: 18, borderRadius: 4, borderWidth: 1, alignItems: "center", justifyContent: "center" },
  channelName: { fontSize: type.body, flex: 1 },
  saveRow: { flexDirection: "row", alignItems: "center", gap: space.sm, marginTop: space.xs },
  saveBtn: { minHeight: 38, paddingHorizontal: space.md },
});
