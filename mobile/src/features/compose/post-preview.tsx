import { Image } from "expo-image";
import { BarChart2, Bookmark, Heart, MessageCircle, Repeat2, Share } from "lucide-react-native";
import { Linking, StyleSheet, Text, View } from "react-native";

import { mediaSource } from "@/data/media";
import { firstURL, type MediaMeta } from "@/data/posts";
import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";

/**
 * A lightweight X-authentic preview of the composed post: avatar, name/handle,
 * body, attached media, and (when there is no media) a link card for the first
 * URL. Mirrors the web composer's preview so authors see what they will publish.
 */
export function PostPreview({
  workspaceId,
  handle,
  body,
  media,
}: {
  workspaceId: string;
  handle: string | null;
  body: string;
  media: MediaMeta[];
}) {
  const { palette } = usePalette();
  const name = handle ? handle.replace(/^@/, "") : "Your account";
  const at = handle ? (handle.startsWith("@") ? handle : `@${handle}`) : "@you";
  const link = media.length === 0 ? firstURL(body) : undefined;
  const domain = link ? safeDomain(link) : null;

  return (
    <View style={[styles.card, { backgroundColor: palette.elevated, borderColor: palette.separator }]}>
      <View style={styles.head}>
        <View style={[styles.avatar, { backgroundColor: palette.accent }]}>
          <Text style={styles.avatarText}>{name.charAt(0).toUpperCase()}</Text>
        </View>
        <View style={{ flex: 1, minWidth: 0 }}>
          <Text style={[styles.name, { color: palette.fg }]} numberOfLines={1}>
            {name}
          </Text>
          <Text style={[styles.handle, { color: palette.fgSubtle }]} numberOfLines={1}>
            {at}
          </Text>
        </View>
      </View>

      {body.trim().length > 0 ? (
        <Text style={[styles.body, { color: palette.fg }]}>{body}</Text>
      ) : (
        <Text style={[styles.body, { color: palette.fgSubtle }]}>Your post text will appear here.</Text>
      )}

      {media.length > 0 && (
        <View style={styles.mediaGrid}>
          {media.slice(0, 4).map((m) =>
            m.kind === "video" ? (
              <View key={m.media_id} style={[styles.mediaCell, styles.video]}>
                <Text style={styles.videoLabel}>video</Text>
              </View>
            ) : (
              <Image
                key={m.media_id}
                source={mediaSource(workspaceId, m.media_id)}
                style={[styles.mediaCell, media.length === 1 && styles.mediaSingle]}
                contentFit="cover"
              />
            ),
          )}
        </View>
      )}

      {link && (
        <Text
          style={[styles.linkCard, { borderColor: palette.separator, color: palette.fgMuted }]}
          onPress={() => Linking.openURL(link)}
        >
          🔗 {domain}
        </Text>
      )}

      <View style={styles.actions}>
        {[MessageCircle, Repeat2, Heart, BarChart2, Bookmark, Share].map((Glyph, i) => (
          <Glyph key={i} size={15} color={palette.fgSubtle} />
        ))}
      </View>
    </View>
  );
}

function safeDomain(url: string): string {
  try {
    return new URL(url).hostname.replace(/^www\./, "");
  } catch {
    return url;
  }
}

const styles = StyleSheet.create({
  card: { borderWidth: StyleSheet.hairlineWidth, borderRadius: radius.lg, padding: space.md, gap: space.sm },
  head: { flexDirection: "row", alignItems: "center", gap: space.sm },
  avatar: { width: 38, height: 38, borderRadius: 19, alignItems: "center", justifyContent: "center" },
  avatarText: { color: "#fff", fontSize: type.subhead, fontWeight: "700" },
  name: { fontSize: type.body, fontWeight: "700" },
  handle: { fontSize: type.caption },
  body: { fontSize: type.subhead, lineHeight: 21 },
  mediaGrid: { flexDirection: "row", flexWrap: "wrap", gap: 2, borderRadius: radius.md, overflow: "hidden" },
  mediaCell: { flexGrow: 1, flexBasis: "48%", aspectRatio: 1, backgroundColor: "#0008" },
  mediaSingle: { flexBasis: "100%", aspectRatio: 16 / 9 },
  video: { alignItems: "center", justifyContent: "center" },
  videoLabel: { color: "#fff", fontSize: type.caption },
  linkCard: { borderWidth: StyleSheet.hairlineWidth, borderRadius: radius.md, padding: space.sm, fontSize: type.caption },
  actions: { flexDirection: "row", justifyContent: "space-between", maxWidth: 280, marginTop: space.xs },
});
