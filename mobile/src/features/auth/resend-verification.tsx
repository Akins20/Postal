import { useEffect, useState } from "react";
import { StyleSheet, Text, View } from "react-native";

import { useResendVerification } from "@/data/auth";
import type { NormalizedError } from "@/lib/api-error";
import { space, type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";

const COOLDOWN_SECONDS = 60;

/**
 * "Resend verification email" button with a cooldown timer. After a send (or a
 * rate-limit response) the button is disabled and counts down, mirroring the
 * web ResendVerification.
 */
export function ResendVerification({ email }: { email: string }) {
  const { palette } = usePalette();
  const resend = useResendVerification();
  const [secondsLeft, setSecondsLeft] = useState(0);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (secondsLeft <= 0) return;
    const timer = setInterval(() => setSecondsLeft((s) => s - 1), 1000);
    return () => clearInterval(timer);
  }, [secondsLeft]);

  const onPress = async () => {
    setError(null);
    try {
      await resend.mutateAsync({ email });
      setSent(true);
    } catch (e) {
      setError((e as NormalizedError).message);
    } finally {
      setSecondsLeft(COOLDOWN_SECONDS);
    }
  };

  const label =
    secondsLeft > 0 ? `Resend in ${secondsLeft}s` : "Resend verification email";

  return (
    <View style={styles.wrap}>
      <Button
        variant="secondary"
        onPress={onPress}
        disabled={secondsLeft > 0 || resend.isPending}
        loading={resend.isPending}
      >
        {label}
      </Button>
      {sent && !error && (
        <Text style={[styles.note, { color: palette.fgMuted }]}>
          Sent. Check your inbox, and your spam folder.
        </Text>
      )}
      {error && <Text style={[styles.note, { color: palette.danger }]}>{error}</Text>}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: space.sm },
  note: { fontSize: type.caption, textAlign: "center" },
});
