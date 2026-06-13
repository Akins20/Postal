import { Link } from "expo-router";
import { useState } from "react";
import { StyleSheet, Text } from "react-native";

import { useRequestReset } from "@/data/auth";
import { AuthScaffold } from "@/features/auth/auth-scaffold";
import type { NormalizedError } from "@/lib/api-error";
import { requestResetSchema } from "@/lib/schemas";
import { type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { FormField } from "@/ui/form-field";

export default function ResetScreen() {
  const { palette } = usePalette();
  const reset = useRequestReset();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [done, setDone] = useState(false);

  const submit = async () => {
    setError(undefined);
    const parsed = requestResetSchema.safeParse({ email });
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message);
      return;
    }
    try {
      await reset.mutateAsync(parsed.data);
      setDone(true);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <AuthScaffold
      title="Reset password"
      subtitle="We'll email you a reset link."
      footer={
        <Link href="/login" style={[styles.foot, { color: palette.accent, fontWeight: "600" }]}>
          Back to sign in
        </Link>
      }
    >
      {done ? (
        <Text accessibilityRole="alert" style={{ color: palette.fgMuted, fontSize: type.body }}>
          If that email is registered, a reset link is on its way. Check your inbox.
        </Text>
      ) : (
        <>
          <FormField
            label="Email"
            value={email}
            onChangeText={setEmail}
            error={error}
            autoCapitalize="none"
            autoComplete="email"
            keyboardType="email-address"
            inputMode="email"
          />
          <Button onPress={submit} loading={reset.isPending}>
            Send reset link
          </Button>
        </>
      )}
    </AuthScaffold>
  );
}

const styles = StyleSheet.create({ foot: { fontSize: type.body, textAlign: "center" } });
