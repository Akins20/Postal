"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { ROLE_LABELS, ROLES, type Capability, type Role } from "@/config/capabilities";
import { useAddMember } from "@/data/workspaces";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { FormField } from "@/ui/primitives/form-field";

import { CapabilityCheckboxes } from "./capability-checkboxes";

const schema = z.object({ email: z.email("Enter a valid email address") });
type Values = z.infer<typeof schema>;

export function AddMemberForm({ workspaceId }: { workspaceId: string }) {
  const add = useAddMember(workspaceId);
  const [role, setRole] = useState<Role>("editor");
  const [caps, setCaps] = useState<Capability[]>([]);
  const [custom, setCustom] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const {
    register,
    handleSubmit,
    reset,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<Values>({ resolver: zodResolver(schema) });

  const onSubmit = handleSubmit(async ({ email }) => {
    setFormError(null);
    try {
      await add.mutateAsync({
        email,
        role: custom ? undefined : role,
        capabilities: custom ? caps : undefined,
      });
      reset();
      setCaps([]);
      setCustom(false);
    } catch (e) {
      const err = e as NormalizedError;
      if (err.fieldErrors.email) setError("email", { message: err.fieldErrors.email });
      else setFormError(err.message);
    }
  });

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
        placeholder="teammate@example.com"
        autoComplete="off"
        error={errors.email?.message}
        {...register("email")}
      />
      {!custom && (
        <label className="flex flex-col gap-1.5">
          <span className="text-fg text-sm font-medium">Role</span>
          <select
            value={role}
            onChange={(e) => setRole(e.target.value as Role)}
            className="border-separator bg-elevated text-fg focus-visible:ring-ring h-10 rounded-md border px-3 text-sm focus-visible:ring-2 focus-visible:outline-none"
          >
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {ROLE_LABELS[r]}
              </option>
            ))}
          </select>
        </label>
      )}
      <button
        type="button"
        onClick={() => setCustom((v) => !v)}
        className="text-accent self-start text-xs font-medium hover:underline"
      >
        {custom ? "Use a role instead" : "Customize permissions"}
      </button>
      {custom && <CapabilityCheckboxes value={caps} onChange={setCaps} />}
      <Button type="submit" disabled={isSubmitting} className="mt-1 self-start">
        {isSubmitting ? "Adding…" : "Add member"}
      </Button>
    </form>
  );
}
