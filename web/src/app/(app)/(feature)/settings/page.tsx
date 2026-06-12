"use client";

import { Settings } from "lucide-react";
import { useState } from "react";

import { MembersPanel } from "@/features/members/members-panel";
import { AccountPanel } from "@/features/settings/account-panel";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { cn } from "@/lib/cn";
import { PageHeader } from "@/ui/page-header";
import { Spinner } from "@/ui/primitives/spinner";

const TABS = [
  { key: "account", label: "Account" },
  { key: "members", label: "Members" },
] as const;

type Tab = (typeof TABS)[number]["key"];

export default function SettingsPage() {
  const { active, isLoading } = useActiveWorkspace();
  const [tab, setTab] = useState<Tab>("account");

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        icon={Settings}
        title="Settings"
        subtitle="Your account, this workspace, and who can do what."
      />

      <div
        role="tablist"
        aria-label="Settings sections"
        className="bg-fg/5 flex w-fit rounded-lg p-0.5"
      >
        {TABS.map((t) => (
          <button
            key={t.key}
            role="tab"
            type="button"
            aria-selected={tab === t.key}
            onClick={() => setTab(t.key)}
            className={cn(
              "focus-visible:ring-ring rounded-md px-4 py-1.5 text-sm transition-colors focus-visible:ring-2 focus-visible:outline-none",
              tab === t.key ? "bg-elevated text-fg font-medium shadow-sm" : "text-fg-muted",
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      {isLoading && (
        <div className="py-10 text-center">
          <Spinner />
        </div>
      )}
      {active && tab === "account" && <AccountPanel workspace={active} />}
      {active && tab === "members" && <MembersPanel workspaceId={active.id} />}
      {!isLoading && !active && <p className="text-fg-muted text-sm">No workspace selected.</p>}
    </div>
  );
}
