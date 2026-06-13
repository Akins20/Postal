import { Image } from "expo-image";
import { X as XIcon } from "lucide-react-native";
import { useState } from "react";
import { ActivityIndicator, Alert, Pressable, ScrollView, StyleSheet, Text, TextInput, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { platformInfo } from "@/config/platforms";
import { useWallet } from "@/data/billing";
import { useChannels, type Channel } from "@/data/channels";
import { mediaSource, useUploadMedia, type Asset } from "@/data/media";
import {
  firstURL,
  useCreatePost,
  useDeletePost,
  usePosts,
  useUpdatePost,
  useValidatePost,
  type MediaMeta,
  type Post,
  type VariantValidation,
} from "@/data/posts";
import { pickMedia } from "@/features/compose/pick-media";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { Panel } from "@/ui/panel";
import { StatusPill } from "@/ui/status-pill";

const MAX_SAFE = Number.MAX_SAFE_INTEGER;

function toMeta(a: Asset): MediaMeta {
  return { media_id: a.id, kind: a.kind as MediaMeta["kind"], mime: a.mime, bytes: a.bytes };
}

export function ComposeScreen() {
  const { palette } = usePalette();
  const insets = useSafeAreaInsets();
  const { active } = useActiveWorkspace();
  const ws = active?.id ?? "";
  const { data: channels = [] } = useChannels(active?.id);
  const { data: wallet } = useWallet(active?.id);

  const [selected, setSelected] = useState<string[]>([]);
  const [body, setBody] = useState("");
  const [media, setMedia] = useState<MediaMeta[]>([]);
  const [postId, setPostId] = useState<string | undefined>();
  const [verdicts, setVerdicts] = useState<VariantValidation[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const create = useCreatePost(ws);
  const update = useUpdatePost(ws);
  const validate = useValidatePost(ws);
  const upload = useUploadMedia(ws);

  const byId = new Map(channels.map((c) => [c.id, c]));
  const selectedChannels = selected.map((id) => byId.get(id)).filter(Boolean) as Channel[];
  const charLimit =
    selectedChannels.length > 0
      ? Math.min(...selectedChannels.map((c) => platformInfo(c.platform).charLimit ?? MAX_SAFE))
      : undefined;
  const remaining = charLimit !== undefined && charLimit !== MAX_SAFE ? charLimit - body.length : undefined;
  const over = remaining !== undefined && remaining < 0;
  const mediaRequiredBy = [
    ...new Set(selectedChannels.filter((c) => platformInfo(c.platform).requiresMedia).map((c) => platformInfo(c.platform).label)),
  ];
  const missingMedia = mediaRequiredBy.length > 0 && media.length === 0;
  const saving = create.isPending || update.isPending || validate.isPending;

  const toggle = (id: string) => {
    setVerdicts(null);
    setSelected((s) => (s.includes(id) ? s.filter((x) => x !== id) : [...s, id]));
  };

  const attach = async () => {
    setError(null);
    const file = await pickMedia();
    if (!file) return;
    try {
      const asset = await upload.mutateAsync(file);
      setMedia((m) => [...m, toMeta(asset)]);
    } catch (e) {
      setError((e as { message?: string }).message ?? "Upload failed.");
    }
  };

  const save = async () => {
    setError(null);
    setVerdicts(null);
    const variants = selected.map((channel_id) => ({
      channel_id,
      body,
      media: media.length > 0 ? media : undefined,
    }));
    try {
      const post = postId
        ? await update.mutateAsync({ postId, variants })
        : await create.mutateAsync({ variants });
      setPostId(post.id);
      setVerdicts(await validate.mutateAsync({ postId: post.id }));
    } catch (e) {
      setError((e as { message?: string }).message ?? "Save failed.");
    }
  };

  const reset = () => {
    setSelected([]); setBody(""); setMedia([]); setPostId(undefined); setVerdicts(null); setError(null);
  };

  // Pay-per-use cost (X), tiered by content.
  const costs = wallet?.publish_costs;
  const xSelected = selectedChannels.some((c) => platformInfo(c.platform).payPerUse);
  const hasLink = Boolean(firstURL(body));
  const xCost = costs
    ? hasLink
      ? (costs.twitter_url ?? costs.twitter)
      : media.length > 0
        ? (costs.twitter_media ?? costs.twitter)
        : costs.twitter
    : undefined;

  return (
    <ScrollView
      style={{ backgroundColor: palette.surface }}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + space.lg }]}
      keyboardShouldPersistTaps="handled"
    >
      <View style={styles.titleRow}>
        <Text style={[styles.title, { color: palette.fg }]}>Compose</Text>
        {postId && (
          <Button variant="ghost" onPress={reset} style={styles.smallBtn}>
            New post
          </Button>
        )}
      </View>

      <Panel>
        <Text style={[styles.cardTitle, { color: palette.fg }]}>Publish to</Text>
        {channels.length === 0 && (
          <Text style={[styles.sub, { color: palette.fgMuted }]}>
            Connect a channel first on the Channels tab.
          </Text>
        )}
        <View style={styles.chips}>
          {channels.map((c) => {
            const on = selected.includes(c.id);
            const disabled = c.status !== "active";
            return (
              <Pressable
                key={c.id}
                accessibilityRole="checkbox"
                accessibilityState={{ checked: on, disabled }}
                accessibilityLabel={`@${c.handle}`}
                disabled={disabled}
                onPress={() => toggle(c.id)}
                style={[
                  styles.chip,
                  {
                    borderColor: on ? palette.accent : palette.separator,
                    backgroundColor: on ? `${palette.accent}1f` : "transparent",
                    opacity: disabled ? 0.5 : 1,
                  },
                ]}
              >
                {(() => {
                  const G = platformInfo(c.platform).Glyph;
                  return <G size={14} color={palette.fg} />;
                })()}
                <Text style={[styles.chipText, { color: palette.fg }]}>@{c.handle}</Text>
              </Pressable>
            );
          })}
        </View>
      </Panel>

      <Panel>
        <View style={styles.editorHead}>
          <Text style={[styles.cardTitle, { color: palette.fg, marginBottom: 0 }]}>Post text</Text>
          {remaining !== undefined && (
            <Text style={[styles.counter, { color: over ? palette.danger : palette.fgSubtle }]}>
              {over ? `${-remaining} over` : `${remaining} left`}
            </Text>
          )}
        </View>
        <TextInput
          accessibilityLabel="Post text"
          value={body}
          onChangeText={(t) => { setBody(t); setVerdicts(null); }}
          placeholder="What do you want to share?"
          placeholderTextColor={palette.fgSubtle}
          multiline
          style={[
            styles.editor,
            { color: palette.fg, backgroundColor: palette.surface, borderColor: over ? palette.danger : palette.separator },
          ]}
        />

        <View style={styles.mediaStrip}>
          {media.map((m) => (
            <View key={m.media_id} style={styles.thumbWrap}>
              {m.kind === "video" ? (
                <View style={[styles.thumb, { backgroundColor: palette.fg }]}>
                  <Text style={styles.videoTag}>video</Text>
                </View>
              ) : (
                <Image source={mediaSource(ws, m.media_id)} style={styles.thumb} contentFit="cover" />
              )}
              <Pressable
                accessibilityLabel="Remove attachment"
                onPress={() => setMedia((cur) => cur.filter((x) => x.media_id !== m.media_id))}
                style={[styles.thumbRemove, { backgroundColor: palette.fg }]}
              >
                <XIcon size={12} color={palette.surface} />
              </Pressable>
            </View>
          ))}
          <Button variant="secondary" onPress={attach} loading={upload.isPending} style={styles.attachBtn}>
            Add media
          </Button>
        </View>

        {xSelected && xCost !== undefined && xCost > 0 && (
          <Text style={[styles.notice, { color: palette.fg, backgroundColor: `${palette.warning}1a`, borderColor: `${palette.warning}40` }]}>
            Publishing to X costs {xCost} credits per channel for {hasLink ? "link posts" : media.length > 0 ? "media posts" : "plain posts"}. Other platforms are free.
          </Text>
        )}
        {missingMedia && (
          <Text style={[styles.warn, { color: palette.warning }]}>
            {mediaRequiredBy.join(" and ")} need an image or video. Attach media or deselect.
          </Text>
        )}
        {error && (
          <Text accessibilityRole="alert" style={[styles.warn, { color: palette.danger }]}>{error}</Text>
        )}

        {verdicts && (
          <View style={styles.verdicts}>
            <Text style={[styles.cardTitle, { color: palette.fg }]}>Draft saved</Text>
            {verdicts.map((v) => {
              const ch = byId.get(v.channel_id);
              return (
                <View key={v.channel_id} style={styles.verdictRow}>
                  <StatusPill tone={v.valid ? "success" : "danger"}>{v.valid ? "Ready" : "Needs changes"}</StatusPill>
                  <Text style={[styles.sub, { color: palette.fgMuted, flex: 1 }]} numberOfLines={2}>
                    {ch ? `@${ch.handle}` : v.channel_id.slice(0, 8)}
                    {!v.valid && v.message ? ` - ${v.message}` : ""}
                  </Text>
                </View>
              );
            })}
          </View>
        )}

        <Button
          onPress={save}
          loading={saving}
          disabled={selected.length === 0 || !body.trim() || missingMedia}
          style={{ marginTop: space.md }}
        >
          {postId ? "Update draft" : "Save draft"}
        </Button>
      </Panel>

      <DraftsCard workspaceId={ws} onEdit={(p) => {
        setPostId(p.id);
        setBody(p.variants?.[0]?.body ?? "");
        setSelected((p.variants ?? []).map((v) => v.channel_id));
        setMedia(p.variants?.[0]?.media ?? []);
        setVerdicts(null);
      }} />
    </ScrollView>
  );
}

function DraftsCard({ workspaceId, onEdit }: { workspaceId: string; onEdit: (p: Post) => void }) {
  const { palette } = usePalette();
  const { data: posts, isPending } = usePosts(workspaceId);
  const del = useDeletePost(workspaceId);

  const confirmDelete = (id: string) =>
    Alert.alert("Delete this draft?", "This removes the draft and its variants.", [
      { text: "Cancel", style: "cancel" },
      { text: "Delete", style: "destructive", onPress: () => del.mutate({ postId: id }) },
    ]);

  return (
    <Panel>
      <Text style={[styles.cardTitle, { color: palette.fg }]}>Your posts</Text>
      {isPending && <ActivityIndicator color={palette.fgSubtle} />}
      {posts?.length === 0 && (
        <Text style={[styles.sub, { color: palette.fgMuted }]}>Nothing saved yet.</Text>
      )}
      {posts?.map((p) => (
        <View key={p.id} style={[styles.draftRow, { borderBottomColor: palette.separator }]}>
          <Pressable style={{ flex: 1 }} onPress={() => onEdit(p)}>
            <Text style={[styles.sub, { color: palette.fg }]} numberOfLines={1}>
              {p.variants?.[0]?.body || "Saved post"}
            </Text>
            <Text style={[styles.draftMeta, { color: palette.fgSubtle }]}>
              {p.status} · {new Date(p.created_at).toLocaleDateString()}
            </Text>
          </Pressable>
          <Button variant="ghost" onPress={() => confirmDelete(p.id)} style={styles.smallBtn}>
            Delete
          </Button>
        </View>
      ))}
    </Panel>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.lg, paddingBottom: space.xxl * 2 },
  titleRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  title: { fontSize: type.display, fontWeight: "700", letterSpacing: -0.5 },
  cardTitle: { fontSize: type.body, fontWeight: "600", marginBottom: space.sm },
  sub: { fontSize: type.caption + 1 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: space.sm },
  chip: { flexDirection: "row", alignItems: "center", gap: 6, borderWidth: 1, borderRadius: radius.full, paddingHorizontal: space.md, paddingVertical: space.xs + 2 },
  chipText: { fontSize: type.caption + 1, fontWeight: "500" },
  editorHead: { flexDirection: "row", justifyContent: "space-between", alignItems: "baseline", marginBottom: space.xs },
  counter: { fontSize: type.caption, fontVariant: ["tabular-nums"] },
  editor: { minHeight: 120, borderWidth: StyleSheet.hairlineWidth, borderRadius: radius.md, padding: space.md, fontSize: type.subhead, textAlignVertical: "top" },
  mediaStrip: { flexDirection: "row", flexWrap: "wrap", gap: space.sm, marginTop: space.md, alignItems: "center" },
  thumbWrap: { position: "relative" },
  thumb: { width: 64, height: 64, borderRadius: radius.md, alignItems: "center", justifyContent: "center" },
  videoTag: { color: "#fff", fontSize: 10 },
  thumbRemove: { position: "absolute", top: -6, right: -6, width: 20, height: 20, borderRadius: 10, alignItems: "center", justifyContent: "center" },
  attachBtn: { minHeight: 40, paddingHorizontal: space.md },
  notice: { fontSize: type.caption, marginTop: space.md, padding: space.sm, borderRadius: radius.md, borderWidth: 1 },
  warn: { fontSize: type.caption, marginTop: space.sm },
  verdicts: { marginTop: space.md, gap: space.sm },
  verdictRow: { flexDirection: "row", alignItems: "center", gap: space.sm },
  draftRow: { flexDirection: "row", alignItems: "center", gap: space.sm, paddingVertical: space.md, borderBottomWidth: StyleSheet.hairlineWidth },
  draftMeta: { fontSize: type.caption, marginTop: 2 },
  smallBtn: { minHeight: 32, paddingHorizontal: space.sm },
});
