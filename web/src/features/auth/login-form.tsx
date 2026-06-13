"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { useLogin } from "@/data/auth";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { FormField } from "@/ui/primitives/form-field";

import { applyServerErrors } from "./form-errors";
import { ResendVerification } from "./resend-verification";
import { loginSchema, type LoginValues } from "./schemas";

export function LoginForm() {
  const router = useRouter();
  const login = useLogin();
  const [formError, setFormError] = useState<string | null>(null);
  // When set, the account exists but its email is unverified: show a resend
  // affordance instead of a dead-end error.
  const [unverifiedEmail, setUnverifiedEmail] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<LoginValues>({ resolver: zodResolver(loginSchema) });

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null);
    try {
      await login.mutateAsync(values);
      router.replace("/");
    } catch (e) {
      const err = e as NormalizedError;
      if (err.code === "email_not_verified") {
        setUnverifiedEmail(values.email);
        return;
      }
      if (!applyServerErrors(err, setError)) {
        setFormError(err.message);
      }
    }
  });

  if (unverifiedEmail) {
    return (
      <div role="status" className="flex flex-col gap-4 text-center">
        <p className="text-fg-muted text-sm">
          Please verify your email before signing in. We sent a link to{" "}
          <span className="text-fg font-medium">{unverifiedEmail}</span>.
        </p>
        <ResendVerification email={unverifiedEmail} />
        <Button variant="secondary" onClick={() => setUnverifiedEmail(null)}>
          Back to sign in
        </Button>
      </div>
    );
  }

  return (
    <form onSubmit={onSubmit} noValidate className="flex flex-col gap-4">
      {formError && (
        <p role="alert" className="bg-danger/10 text-danger rounded-md px-3 py-2 text-sm">
          {formError}
        </p>
      )}
      <FormField
        label="Email"
        type="email"
        autoComplete="email"
        error={errors.email?.message}
        {...register("email")}
      />
      <FormField
        label="Password"
        type="password"
        autoComplete="current-password"
        error={errors.password?.message}
        {...register("password")}
      />
      <Button type="submit" disabled={isSubmitting} className="mt-1">
        {isSubmitting ? "Signing in…" : "Sign in"}
      </Button>
    </form>
  );
}
