import { Link, useRouter } from "expo-router";
import { useState } from "react";
import { StyleSheet, Text } from "react-native";

import { useSignup } from "@/data/auth";
import { AuthScaffold } from "@/features/auth/auth-scaffold";
import { ResendVerification } from "@/features/auth/resend-verification";
import type { NormalizedError } from "@/lib/api-error";
import { signupSchema } from "@/lib/schemas";
import { type } from "@/lib/tokens";
import { usePalette } from "@/lib/use-palette";
import { Button } from "@/ui/button";
import { FormField } from "@/ui/form-field";

export default function SignupScreen() {
  const router = useRouter();
  const { palette } = usePalette();
  const signup = useSignup();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string }>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [sentEmail, setSentEmail] = useState<string | null>(null);

  const submit = async () => {
    setFormError(null);
    const parsed = signupSchema.safeParse({ email, password });
    if (!parsed.success) {
      const fe: typeof errors = {};
      for (const issue of parsed.error.issues) fe[issue.path[0] as "email" | "password"] = issue.message;
      setErrors(fe);
      return;
    }
    setErrors({});
    try {
      await signup.mutateAsync(parsed.data);
      // No session is issued; the user must verify their email, then sign in.
      setSentEmail(parsed.data.email);
    } catch (e) {
      setFormError((e as NormalizedError).message);
    }
  };

  if (sentEmail) {
    return (
      <AuthScaffold
        title="Check your email"
        subtitle={`We sent a verification link to ${sentEmail}.`}
      >
        <Text style={[styles.foot, { color: palette.fgMuted }]}>
          Open the link to finish setting up your account, then sign in.
        </Text>
        <ResendVerification email={sentEmail} />
        <Button variant="secondary" onPress={() => router.replace("/login")}>
          Go to sign in
        </Button>
      </AuthScaffold>
    );
  }

  return (
    <AuthScaffold
      title="Create your account"
      subtitle="Free, no paywall."
      footer={
        <Text style={[styles.foot, { color: palette.fgMuted }]}>
          Already have an account?{" "}
          <Link href="/login" style={{ color: palette.accent, fontWeight: "600" }}>
            Sign in
          </Link>
        </Text>
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
        hint="At least 8 characters."
        secureTextEntry
        autoComplete="new-password"
      />
      <Button onPress={submit} loading={signup.isPending}>
        Create account
      </Button>
    </AuthScaffold>
  );
}

const styles = StyleSheet.create({ foot: { fontSize: type.body, textAlign: "center" } });
