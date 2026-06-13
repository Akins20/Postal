import { Link, useRouter } from "expo-router";
import { useState } from "react";
import { StyleSheet, Text } from "react-native";

import { useLogin } from "@/data/auth";
import { AuthScaffold } from "@/features/auth/auth-scaffold";
import { ResendVerification } from "@/features/auth/resend-verification";
import type { NormalizedError } from "@/lib/api-error";
import { loginSchema } from "@/lib/schemas";
import { type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { FormField } from "@/ui/form-field";

export default function LoginScreen() {
  const router = useRouter();
  const { palette } = usePalette();
  const login = useLogin();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [unverifiedEmail, setUnverifiedEmail] = useState<string | null>(null);

  const submit = async () => {
    setFormError(null);
    const parsed = loginSchema.safeParse({ email, password });
    if (!parsed.success) {
      const fe: typeof errors = {};
      for (const issue of parsed.error.issues) fe[issue.path[0] as "email" | "password"] = issue.message;
      setErrors(fe);
      return;
    }
    setErrors({});
    try {
      await login.mutateAsync(parsed.data);
      router.replace("/");
    } catch (e) {
      const err = e as NormalizedError;
      if (err.code === "email_not_verified") {
        setUnverifiedEmail(parsed.data.email);
        return;
      }
      setFormError(err.message);
    }
  };

  if (unverifiedEmail) {
    return (
      <AuthScaffold title="Verify your email" subtitle="Please verify before signing in.">
        <Text style={[styles.foot, { color: palette.fgMuted }]}>
          We sent a verification link to {unverifiedEmail}.
        </Text>
        <ResendVerification email={unverifiedEmail} />
        <Button variant="secondary" onPress={() => setUnverifiedEmail(null)}>
          Back to sign in
        </Button>
      </AuthScaffold>
    );
  }

  return (
    <AuthScaffold
      title="Sign in"
      subtitle="Welcome back."
      footer={
        <>
          <Text style={[styles.foot, { color: palette.fgMuted }]}>
            New here?{" "}
            <Link href="/signup" style={{ color: palette.accent, fontWeight: "600" }}>
              Create an account
            </Link>
          </Text>
          <Link href="/reset" style={[styles.foot, { color: palette.fgSubtle }]}>
            Forgot your password?
          </Link>
        </>
      }
    >
      {formError && (
        <Text accessibilityRole="alert" style={{ color: palette.danger, fontSize: type.body }}>
          {formError}
        </Text>
      )}
      <FormField
        label="Email"
        value={email}
        onChangeText={setEmail}
        error={errors.email}
        autoCapitalize="none"
        autoComplete="email"
        keyboardType="email-address"
        inputMode="email"
      />
      <FormField
        label="Password"
        value={password}
        onChangeText={setPassword}
        error={errors.password}
        secureTextEntry
        autoComplete="password"
      />
      <Button onPress={submit} loading={login.isPending}>
        Sign in
      </Button>
    </AuthScaffold>
  );
}

const styles = StyleSheet.create({ foot: { fontSize: type.body, textAlign: "center" } });
