"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { useRequestReset } from "@/data/auth";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { FormField } from "@/ui/primitives/form-field";

import { requestResetSchema, type RequestResetValues } from "./schemas";

export function RequestResetForm() {
  const request = useRequestReset();
  const [formError, setFormError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RequestResetValues>({ resolver: zodResolver(requestResetSchema) });

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null);
    try {
      await request.mutateAsync(values);
      setDone(true);
    } catch (e) {
      setFormError((e as NormalizedError).message);
    }
  });

  if (done) {
    return (
      <p role="status" className="text-fg-muted text-center text-sm">
        If that email is registered, we&apos;ve sent a password-reset link. Check your inbox.
      </p>
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
      <Button type="submit" disabled={isSubmitting} className="mt-1">
        {isSubmitting ? "Sending…" : "Send reset link"}
      </Button>
    </form>
  );
}
