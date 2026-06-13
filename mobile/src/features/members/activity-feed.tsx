import { ActivityIndicator, StyleSheet, Text, View } from "react-native";

import { useActivity, type ActivityEntry } from "@/data/governance";
import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Panel } from "@/ui/panel";

const ACTION_LABELS: Record<string, string> = {
  "user.login": "signed in",
  "user.signup": "signed up",
  "user.email_verified": "verified their email",
  "post.schedule": "scheduled a post",
  "post.schedule_slots": "queued a post to slots",
  "channel.connected": "connected a channel",
  "channel.disconnected": "disconnected a channel",
  "member.added": "added a member",
  "member.capabilities_updated": "changed a member's permissions",
};

function label(action: string): string {
  return ACTION_LABELS[action] ?? action.replace(/[._]/g, " ");
}

function Row({ entry }: { entry: ActivityEntry }) {
  const { palette } = usePalette();
  return (
    <View style={[styles.row, { borderBottomColor: palette.separator }]}>
      <Text style={[styles.text, { color: palette.fg }]} numberOfLines={2}>
        <Text style={{ fontWeight: "600" }}>{entry.actor_email || "System"}</Text> {label(entry.action)}
        {entry.target ? ` · ${entry.target}` : ""}
      </Text>
      <Text style={[styles.time, { color: palette.fgSubtle }]}>
        {new Date(entry.created_at).toLocaleString()}
      </Text>
    </View>
  );
}

/** "Who did what" feed for the workspace. */
export function ActivityFeed({ workspaceId }: { workspaceId: string }) {
  const { palette } = usePalette();
  const { data: activity, isPending, isError } = useActivity(workspaceId);
  return (
    <Panel>
      <Text style={[styles.title, { color: palette.fg }]}>Activity</Text>
      {isPending && <ActivityIndicator color={palette.fgSubtle} />}
      {isError && (
        <Text accessibilityRole="alert" style={[styles.time, { color: palette.danger }]}>
          Couldn&apos;t load activity.
        </Text>
      )}
      {activity?.length === 0 && (
        <Text style={[styles.time, { color: palette.fgMuted, paddingVertical: space.sm }]}>
          No activity yet.
        </Text>
      )}
      {activity?.map((e) => <Row key={e.id} entry={e} />)}
    </Panel>
  );
}

const styles = StyleSheet.create({
  title: { fontSize: type.body, fontWeight: "600", marginBottom: space.sm },
  row: { paddingVertical: space.sm, borderBottomWidth: StyleSheet.hairlineWidth, gap: 2 },
  text: { fontSize: type.body },
  time: { fontSize: type.caption },
});
