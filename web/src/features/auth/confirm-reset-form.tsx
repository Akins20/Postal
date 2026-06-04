"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useState } from "react";
import { useForm } from "react-hook-form";

import { useConfirmReset } from "@/data/auth";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { FormField } from "@/ui/primitives/form-field";

import { applyServerErrors } from "./form-errors";
import { confirmResetSchema, type ConfirmResetValues } from "./schemas";

export function ConfirmResetForm({ token }: { token: string }) {
  const confirm = useConfirmReset();
  const [formError, setFormError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ConfirmResetValues>({ resolver: zodResolver(confirmResetSchema) });

  if (!token) {
    return (
      <p role="alert" className="text-danger text-center text-sm">
        This reset link is invalid or missing its token.
      </p>
    );
  }

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null);
    try {
      await confirm.mutateAsync({ token, new_password: values.new_password });
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
        <p className="text-fg-muted text-sm">Your password has been updated.</p>
        <Button asChild>
          <Link href="/login">Sign in</Link>
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
        label="New password"
        type="password"
        autoComplete="new-password"
        hint="At least 8 characters."
        error={errors.new_password?.message}
        {...register("new_password")}
      />
      <Button type="submit" disabled={isSubmitting} className="mt-1">
        {isSubmitting ? "Updating…" : "Update password"}
      </Button>
    </form>
  );
}
