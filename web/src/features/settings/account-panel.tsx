"use client";

import { format } from "date-fns";

import { useMe } from "@/data/auth";
import type { Workspace } from "@/data/workspaces";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";
import { StatusPill } from "@/ui/primitives/status-pill";
import { ThemeToggle } from "@/ui/theme-toggle";

/** Account, appearance, and workspace facts on the Settings screen. */
export function AccountPanel({ workspace }: { workspace: Workspace }) {
  const { data: user, isPending } = useMe();

  return (
    <div className="flex flex-col gap-6">
      <Panel className="p-6">
        <h2 className="text-fg text-sm font-semibold">Account</h2>
        <p className="text-fg-muted mt-1 mb-4 text-sm">Who you&apos;re signed in as.</p>
        {isPending && <Spinner label="Loading account" />}
        {user && (
          <dl className="flex flex-col gap-3 text-sm">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <dt className="text-fg-muted">Email</dt>
              <dd className="text-fg flex items-center gap-2">
                {user.email}
                <StatusPill tone={user.email_verified ? "success" : "warning"}>
                  {user.email_verified ? "Verified" : "Unverified"}
                </StatusPill>
              </dd>
            </div>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <dt className="text-fg-muted">Member since</dt>
              <dd className="text-fg">{format(new Date(user.created_at), "d MMMM yyyy")}</dd>
            </div>
          </dl>
        )}
      </Panel>

      <Panel className="p-6">
        <h2 className="text-fg text-sm font-semibold">Appearance</h2>
        <div className="mt-3 flex items-center justify-between gap-3">
          <p className="text-fg-muted text-sm">
            Light or dark — follows your system until you choose.
          </p>
          <ThemeToggle />
        </div>
      </Panel>

      <Panel className="p-6">
        <h2 className="text-fg text-sm font-semibold">Workspace</h2>
        <dl className="mt-3 flex flex-col gap-3 text-sm">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <dt className="text-fg-muted">Name</dt>
            <dd className="text-fg">{workspace.name}</dd>
          </div>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <dt className="text-fg-muted">Plan</dt>
            <dd>
              <StatusPill tone="accent">{workspace.plan}</StatusPill>
            </dd>
          </div>
        </dl>
      </Panel>
    </div>
  );
}
