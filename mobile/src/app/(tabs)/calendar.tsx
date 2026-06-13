import { useState } from "react";
import { ActivityIndicator, Alert, ScrollView, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";

import { useChannels } from "@/data/channels";
import { useCalendar, useCancelJob, type Job } from "@/data/schedule";
import { JOB_TONE } from "@/features/schedule/job-status";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { Panel } from "@/ui/panel";
import { StatusPill } from "@/ui/status-pill";

function dayKey(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric" });
}

function JobRow({ workspaceId, job, handle }: { workspaceId: string; job: Job; handle: string }) {
  const { palette } = usePalette();
  const cancel = useCancelJob(workspaceId);
  const confirm = () =>
    Alert.alert("Cancel this scheduled post?", `It won't be published to @${handle}.`, [
      { text: "Keep", style: "cancel" },
      { text: "Cancel job", style: "destructive", onPress: () => cancel.mutate({ jobId: job.id }) },
    ]);

  return (
    <View style={[styles.jobRow, { borderBottomColor: palette.separator }]}>
      <Text style={[styles.time, { color: palette.fg }]}>
        {new Date(job.run_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
      </Text>
      <Text style={[styles.handle, { color: palette.fgMuted }]} numberOfLines={1}>
        @{handle}
        {job.status === "failed" && job.last_error ? ` - ${job.last_error}` : ""}
      </Text>
      <StatusPill tone={JOB_TONE[job.status]}>{job.status}</StatusPill>
      {job.status === "scheduled" && (
        <Button variant="ghost" onPress={confirm} loading={cancel.isPending} style={styles.cancelBtn}>
          Cancel
        </Button>
      )}
    </View>
  );
}

export default function CalendarScreen() {
  const { palette } = usePalette();
  const insets = useSafeAreaInsets();
  const { active } = useActiveWorkspace();
  // Window: a little in the past (to show just-run jobs) through 30 days out.
  const [from] = useState(() => new Date(Date.now() - 60 * 60 * 1000).toISOString());
  const [to] = useState(() => new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString());
  const { data: jobs, isPending, isError } = useCalendar(active?.id, from, to);
  const { data: channels = [] } = useChannels(active?.id);
  const handleFor = (channelId: string) =>
    channels.find((c) => c.id === channelId)?.handle ?? channelId.slice(0, 8);

  const sorted = [...(jobs ?? [])].sort((a, b) => a.run_at.localeCompare(b.run_at));
  const groups: Record<string, Job[]> = {};
  for (const j of sorted) (groups[dayKey(j.run_at)] ??= []).push(j);

  return (
    <ScrollView
      style={{ backgroundColor: palette.surface }}
      contentContainerStyle={[styles.content, { paddingTop: insets.top + space.lg }]}
    >
      <Text style={[styles.title, { color: palette.fg }]}>Calendar</Text>
      {(!active || isPending) && <ActivityIndicator color={palette.fgSubtle} />}
      {isError && (
        <Text accessibilityRole="alert" style={[styles.empty, { color: palette.danger }]}>
          Couldn&apos;t load the calendar.
        </Text>
      )}
      {jobs?.length === 0 && (
        <Panel>
          <Text style={[styles.empty, { color: palette.fgMuted }]}>
            Nothing scheduled. Save a draft on the Compose tab, then publish it.
          </Text>
        </Panel>
      )}
      {Object.entries(groups).map(([day, dayJobs]) => (
        <View key={day} style={{ gap: space.xs }}>
          <Text style={[styles.day, { color: palette.fgMuted }]}>{day}</Text>
          <Panel>
            {dayJobs.map((j) => (
              <JobRow key={j.id} workspaceId={active!.id} job={j} handle={handleFor(j.channel_id)} />
            ))}
          </Panel>
        </View>
      ))}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: space.lg, gap: space.lg, paddingBottom: space.xxl * 2 },
  title: { fontSize: type.display, fontWeight: "700", letterSpacing: -0.5 },
  day: { fontSize: type.caption, fontWeight: "700", textTransform: "uppercase", letterSpacing: 0.4, marginTop: space.sm },
  jobRow: { flexDirection: "row", alignItems: "center", gap: space.sm, paddingVertical: space.md, borderBottomWidth: StyleSheet.hairlineWidth },
  time: { fontSize: type.body, fontWeight: "600", width: 64, fontVariant: ["tabular-nums"] },
  handle: { fontSize: type.caption + 1, flex: 1 },
  empty: { fontSize: type.body },
  cancelBtn: { minHeight: 32, paddingHorizontal: space.sm },
});
