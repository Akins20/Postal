import { ActivityIndicator, ScrollView, StyleSheet, Text, View } from "react-native";

import { useAnalyticsOverview } from "@/data/analytics";
import { useChannels } from "@/data/channels";
import { usePosts } from "@/data/posts";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Panel } from "@/ui/panel";

export default function AnalyticsScreen() {
  const { palette } = usePalette();
  const { active } = useActiveWorkspace();
  const { data: rows, isPending } = useAnalyticsOverview(active?.id);
  const { data: channels = [] } = useChannels(active?.id);
  const { data: posts = [] } = usePosts(active?.id);

  const handleFor = (id: string) => channels.find((c) => c.id === id)?.handle ?? id.slice(0, 8);
  const excerptFor = (id: string) => posts.find((p) => p.id === id)?.variants?.[0]?.body;
  const anyPublished = posts.some((p) => p.status === "published");

  return (
    <ScrollView
      style={{ backgroundColor: palette.surface }}
      contentContainerStyle={styles.content}
    >
      {(!active || isPending) && <ActivityIndicator color={palette.fgSubtle} />}
      {rows?.length === 0 && (
        <Panel>
          <Text style={[styles.empty, { color: palette.fg, fontWeight: "600" }]}>
            {anyPublished ? "Metrics are on the way" : "No published posts yet"}
          </Text>
          <Text style={[styles.sub, { color: palette.fgMuted, marginTop: space.xs }]}>
            {anyPublished
              ? "First numbers are captured within about 15 minutes of publishing, then refresh periodically."
              : "Publish a post from the Compose tab first. Metrics appear here within about 15 minutes."}
          </Text>
        </Panel>
      )}
      {rows?.map((row) => (
        <Panel key={`${row.post_id}-${row.channel_id}`}>
          <View style={styles.head}>
            <Text style={[styles.handle, { color: palette.fg }]}>@{handleFor(row.channel_id)}</Text>
            <Text style={[styles.sub, { color: palette.fgSubtle }]}>
              {new Date(row.captured_at).toLocaleDateString()}
            </Text>
          </View>
          {excerptFor(row.post_id) && (
            <Text style={[styles.sub, { color: palette.fgMuted, marginBottom: space.sm }]} numberOfLines={1}>
              {excerptFor(row.post_id)}
            </Text>
          )}
          <View style={styles.metrics}>
            {Object.entries(row.metrics)
              .sort(([a], [b]) => a.localeCompare(b))
              .map(([name, value]) => (
                <View key={name} style={styles.metric}>
                  <Text style={[styles.metricVal, { color: palette.fg }]}>{value}</Text>
                  <Text style={[styles.metricName, { color: palette.fgSubtle }]}>{name}</Text>
                </View>
              ))}
          </View>
        </Panel>
      ))}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.md },
  empty: { fontSize: type.body },
  sub: { fontSize: type.caption + 1 },
  head: { flexDirection: "row", justifyContent: "space-between", alignItems: "baseline" },
  handle: { fontSize: type.body, fontWeight: "600" },
  metrics: { flexDirection: "row", flexWrap: "wrap", gap: space.xl },
  metric: { alignItems: "flex-start" },
  metricVal: { fontSize: type.title, fontWeight: "700", fontVariant: ["tabular-nums"] },
  metricName: { fontSize: type.caption, textTransform: "capitalize" },
});
