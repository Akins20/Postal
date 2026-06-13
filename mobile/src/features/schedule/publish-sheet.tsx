import DateTimePicker from "@react-native-community/datetimepicker";
import { useState } from "react";
import { Modal, Pressable, StyleSheet, Text, View } from "react-native";

import { useSchedulePost } from "@/data/schedule";
import type { NormalizedError } from "@/lib/api-error";
import { radius, space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";

type Mode = "now" | "slots" | "time";

/**
 * Bottom-sheet to publish a saved draft: now, into each channel's next open
 * slot, or at a specific local time (native picker, sent as UTC). "Now" rides
 * the same queue (run_at = now+3s) so retries / wallet gate still apply.
 */
export function PublishSheet({
  workspaceId,
  postId,
  visible,
  onClose,
}: {
  workspaceId: string;
  postId: string;
  visible: boolean;
  onClose: () => void;
}) {
  const { palette } = usePalette();
  const schedule = useSchedulePost(workspaceId);
  const [mode, setMode] = useState<Mode>("now");
  const [when, setWhen] = useState(() => new Date(Date.now() + 60 * 60 * 1000));
  const [picker, setPicker] = useState<"date" | "time" | null>(null);
  const [error, setError] = useState<NormalizedError | null>(null);
  const [scheduled, setScheduled] = useState<number | null>(null);

  const reset = () => {
    setMode("now"); setError(null); setScheduled(null); setPicker(null);
  };
  const close = () => { reset(); onClose(); };

  const submit = async () => {
    setError(null);
    try {
      const jobs = await schedule.mutateAsync(
        mode === "slots"
          ? { postId, toSlots: true }
          : { postId, runAt: mode === "now" ? new Date(Date.now() + 3000).toISOString() : when.toISOString() },
      );
      setScheduled(jobs.length);
    } catch (e) {
      setError(e as NormalizedError);
    }
  };

  // A plain render helper (not a component) so React Compiler doesn't see a new
  // component type created during render.
  const option = (value: Mode, title: string, subtitle: string) => (
    <Pressable
      key={value}
      accessibilityRole="radio"
      accessibilityState={{ selected: mode === value }}
      onPress={() => setMode(value)}
      style={[
        styles.option,
        { borderColor: mode === value ? palette.accent : palette.separator, backgroundColor: mode === value ? `${palette.accent}14` : "transparent" },
      ]}
    >
      <View style={[styles.radio, { borderColor: mode === value ? palette.accent : palette.fgSubtle }]}>
        {mode === value && <View style={[styles.radioDot, { backgroundColor: palette.accent }]} />}
      </View>
      <View style={{ flex: 1 }}>
        <Text style={[styles.optTitle, { color: palette.fg }]}>{title}</Text>
        <Text style={[styles.optSub, { color: palette.fgMuted }]}>{subtitle}</Text>
      </View>
    </Pressable>
  );

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={close}>
      <Pressable style={styles.backdrop} onPress={close} />
      <View style={[styles.sheet, { backgroundColor: palette.elevated, borderColor: palette.separator }]}>
        <Text style={[styles.title, { color: palette.fg }]}>Publish post</Text>

        {scheduled !== null ? (
          <View style={{ gap: space.md }}>
            <Text style={[styles.optSub, { color: palette.fg }]}>
              {scheduled} job{scheduled === 1 ? "" : "s"} created. Track them on the Calendar.
            </Text>
            <Button onPress={close}>Done</Button>
          </View>
        ) : (
          <>
            {option("now", "Publish now", "Goes out within seconds.")}
            {option("slots", "Next open slots", "Use each channel's posting schedule.")}
            {option("time", "Specific time", `In your timezone (${Intl.DateTimeFormat().resolvedOptions().timeZone}).`)}

            {mode === "time" && (
              <View style={styles.timeRow}>
                <Button variant="secondary" onPress={() => setPicker("date")} style={{ flex: 1 }}>
                  {when.toLocaleDateString()}
                </Button>
                <Button variant="secondary" onPress={() => setPicker("time")} style={{ flex: 1 }}>
                  {when.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                </Button>
              </View>
            )}
            {picker && (
              <DateTimePicker
                value={when}
                mode={picker}
                onChange={(_, d) => {
                  setPicker(null);
                  if (d) setWhen(d);
                }}
              />
            )}

            {error && (
              <Text accessibilityRole="alert" style={[styles.error, { color: palette.danger }]}>
                {error.message}
                {error.code === "insufficient_credits" ? " Top up on the Wallet tab." : ""}
              </Text>
            )}

            <View style={styles.actions}>
              <Button variant="secondary" onPress={close} style={{ flex: 1 }}>Cancel</Button>
              <Button onPress={submit} loading={schedule.isPending} style={{ flex: 1 }}>
                {mode === "now" ? "Publish now" : "Schedule"}
              </Button>
            </View>
          </>
        )}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: "rgba(0,0,0,0.45)" },
  sheet: { position: "absolute", bottom: 0, left: 0, right: 0, padding: space.lg, paddingBottom: space.xxl, borderTopLeftRadius: radius.xl, borderTopRightRadius: radius.xl, borderTopWidth: StyleSheet.hairlineWidth, gap: space.md },
  title: { fontSize: type.subhead, fontWeight: "700" },
  option: { flexDirection: "row", alignItems: "center", gap: space.md, borderWidth: 1, borderRadius: radius.md, padding: space.md },
  radio: { width: 20, height: 20, borderRadius: 10, borderWidth: 2, alignItems: "center", justifyContent: "center" },
  radioDot: { width: 10, height: 10, borderRadius: 5 },
  optTitle: { fontSize: type.body, fontWeight: "600" },
  optSub: { fontSize: type.caption + 1 },
  timeRow: { flexDirection: "row", gap: space.sm },
  error: { fontSize: type.caption + 1 },
  actions: { flexDirection: "row", gap: space.sm, marginTop: space.xs },
});
