"use client";

import { useActivity, type ActivityEntry } from "@/data/governance";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";

// Map raw audit action codes to readable phrases. Unknown actions fall back to
// the code with separators tidied up.
const ACTION_LABELS: Record<string, string> = {
  "user.login": "signed in",
  "user.signup": "signed up",
  "user.password_reset_requested": "requested a password reset",
  "user.email_verified": "verified their email",
  "post.schedule": "scheduled a post",
  "post.schedule_slots": "queued a post to slots",
  "channel.connected": "connected a channel",
  "channel.disconnected": "disconnected a channel",
  "member.added": "added a member",
  "member.capabilities_updated": "changed a member's permissions",
};

function label(action: string): string {
  return ACTION_LABELS[action] ?? action.replace(/[._]/g, " ");
}

function ActivityRow({ entry }: { entry: ActivityEntry }) {
  return (
    <li className="border-separator flex items-start justify-between gap-3 border-b py-2.5 last:border-0">
      <div className="min-w-0">
        <p className="text-fg text-sm">
          <span className="font-medium">{entry.actor_email || "System"}</span> {label(entry.action)}
          {entry.target ? <span className="text-fg-muted"> · {entry.target}</span> : null}
        </p>
      </div>
      <span className="text-fg-subtle shrink-0 text-xs">
        {new Date(entry.created_at).toLocaleString()}
      </span>
    </li>
  );
}

/** The "who did what" activity feed for a workspace (admins only). */
export function ActivityFeed({ workspaceId }: { workspaceId: string }) {
  const { data: activity, isPending, isError } = useActivity(workspaceId);

  return (
    <Panel className="p-6">
      <h2 className="text-fg text-sm font-semibold">Activity</h2>
      <p className="text-fg-muted mt-1 mb-3 text-sm">
        Who did what in this workspace, most recent first.
      </p>
      {isPending && (
        <div className="py-6 text-center">
          <Spinner />
        </div>
      )}
      {isError && (
        <p role="alert" className="text-danger text-sm">
          Couldn&apos;t load activity.
        </p>
      )}
      {activity?.length === 0 && <p className="text-fg-muted py-3 text-sm">No activity yet.</p>}
      {activity && activity.length > 0 && (
        <ul className="flex list-none flex-col">
          {activity.map((e) => (
            <ActivityRow key={e.id} entry={e} />
          ))}
        </ul>
      )}
    </Panel>
  );
}
