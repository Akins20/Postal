"use client";

import { MembersPanel } from "@/features/members/members-panel";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { WorkspaceSwitcher } from "@/features/workspace/workspace-switcher";
import { Spinner } from "@/ui/primitives/spinner";

export default function SettingsPage() {
  const { active, isLoading } = useActiveWorkspace();

  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-6 p-6">
      <header className="flex items-center justify-between gap-3">
        <h1 className="text-fg text-lg font-semibold">Settings</h1>
        <WorkspaceSwitcher />
      </header>
      {isLoading && (
        <div className="py-10 text-center">
          <Spinner />
        </div>
      )}
      {active && <MembersPanel workspaceId={active.id} />}
      {!isLoading && !active && <p className="text-fg-muted text-sm">No workspace selected.</p>}
    </div>
  );
}
