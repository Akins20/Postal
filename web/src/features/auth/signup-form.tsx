"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { useSignup } from "@/data/auth";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { FormField } from "@/ui/primitives/form-field";

import { applyServerErrors } from "./form-errors";
import { ResendVerification } from "./resend-verification";
import { signupSchema, type SignupValues } from "./schemas";

export function SignupForm() {
  const signup = useSignup();
  const [formError, setFormError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const [email, setEmail] = useState("");
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<SignupValues>({ resolver: zodResolver(signupSchema) });

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null);
    try {
      await signup.mutateAsync(values);
      setEmail(values.email);
      setDone(true);
    } catch (e) {
      if (!applyServerErrors(e as NormalizedError, setError)) {
        setFormError((e as NormalizedError).message);
      }
    }
  });

  if (done) {
    return (
      <div role="status" className="flex flex-col gap-4 text-center">
        <p className="text-fg-muted text-sm">
          Check your inbox. We sent a verification link to{" "}
          <span className="text-fg font-medium">{email}</span>. Open it to finish setting up your
          account, then sign in.
        </p>
        <ResendVerification email={email} />
        <Button asChild variant="secondary">
          <Link href="/login">Go to sign in</Link>
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
        autoComplete="new-password"
        hint="At least 8 characters."
        error={errors.password?.message}
        {...register("password")}
      />
      <Button type="submit" disabled={isSubmitting} className="mt-1">
        {isSubmitting ? "Creating account…" : "Create account"}
      </Button>
    </form>
  );
}
