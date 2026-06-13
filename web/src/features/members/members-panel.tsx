"use client";

import { ROLE_LABELS, ROLES, type Role } from "@/config/capabilities";
import { useMembers, useUpdateCapabilities, type Member } from "@/data/workspaces";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";
import { StatusPill } from "@/ui/primitives/status-pill";

import { AddMemberForm } from "./add-member-form";
import { ActivityFeed } from "./activity-feed";
import { MemberChannelAccess } from "./member-channel-access";

function MemberRow({ workspaceId, member }: { workspaceId: string; member: Member }) {
  const update = useUpdateCapabilities(workspaceId);
  const count = member.permissions.length;

  return (
    <div className="border-separator border-b py-3 last:border-0">
      <div className="flex items-center justify-between gap-3">
        <div className="flex flex-col">
          <span className="text-fg-muted font-mono text-xs">{member.user_id.slice(0, 8)}…</span>
          <span className="text-fg-subtle text-xs">
            {count} permission{count === 1 ? "" : "s"}
          </span>
        </div>
        {member.role === "owner" ? (
          <StatusPill tone="accent">Owner</StatusPill>
        ) : (
          <select
            aria-label="Member role"
            defaultValue={member.role}
            disabled={update.isPending}
            onChange={(e) =>
              update.mutate({ userId: member.user_id, role: e.target.value as Role })
            }
            className="border-separator bg-elevated text-fg focus-visible:ring-ring h-9 rounded-md border px-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
          >
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {ROLE_LABELS[r]}
              </option>
            ))}
          </select>
        )}
      </div>
      {member.role !== "owner" && (
        <MemberChannelAccess workspaceId={workspaceId} userId={member.user_id} />
      )}
    </div>
  );
}

export function MembersPanel({ workspaceId }: { workspaceId: string }) {
  const { data: members, isPending, isError } = useMembers(workspaceId);

  return (
    <div className="flex flex-col gap-6">
      <Panel className="p-6">
        <h2 className="text-fg text-sm font-semibold">Members</h2>
        <p className="text-fg-muted mt-1 mb-4 text-sm">People with access to this workspace.</p>
        {isPending && (
          <div className="py-6 text-center">
            <Spinner />
          </div>
        )}
        {isError && (
          <p role="alert" className="text-danger text-sm">
            Couldn&apos;t load members. Please try again.
          </p>
        )}
        {members?.length === 0 && <p className="text-fg-muted py-4 text-sm">No members yet.</p>}
        {members?.map((m) => (
          <MemberRow key={m.user_id} workspaceId={workspaceId} member={m} />
        ))}
      </Panel>

      <Panel className="p-6">
        <h2 className="text-fg text-sm font-semibold">Add a member</h2>
        <p className="text-fg-muted mt-1 mb-4 text-sm">
          Invite someone by email and choose what they can do.
        </p>
        <AddMemberForm workspaceId={workspaceId} />
      </Panel>

      <ActivityFeed workspaceId={workspaceId} />
    </div>
  );
}
