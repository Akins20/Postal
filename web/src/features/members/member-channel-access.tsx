"use client";

import { useState } from "react";

import { useChannels } from "@/data/channels";
import { useMemberChannels, useSetMemberChannels, type ChannelAccess } from "@/data/governance";
import type { NormalizedError } from "@/lib/api-error";
import { atHandle } from "@/lib/format";
import { Button } from "@/ui/primitives/button";
import { Spinner } from "@/ui/primitives/spinner";

/** A disclosure that lazily loads + mounts the per-channel publish access editor. */
export function MemberChannelAccess({ workspaceId, userId }: { workspaceId: string; userId: string }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="mt-2">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="text-accent text-xs font-medium hover:underline"
      >
        {open ? "Hide channel access" : "Channel access"}
      </button>
      {open && <AccessLoader workspaceId={workspaceId} userId={userId} />}
    </div>
  );
}

// Loads the current access, then mounts the editor with it as the initial value,
// so the editor can seed state from props (no state-syncing effect).
function AccessLoader({ workspaceId, userId }: { workspaceId: string; userId: string }) {
  const { data: access, isPending } = useMemberChannels(workspaceId, userId);
  if (isPending || !access) {
    return (
      <div className="py-3">
        <Spinner />
      </div>
    );
  }
  return <ChannelAccessEditor workspaceId={workspaceId} userId={userId} initial={access} />;
}

function ChannelAccessEditor({
  workspaceId,
  userId,
  initial,
}: {
  workspaceId: string;
  userId: string;
  initial: ChannelAccess;
}) {
  const { data: channels } = useChannels(workspaceId);
  const save = useSetMemberChannels(workspaceId, userId);
  const [restricted, setRestricted] = useState(initial.restricted);
  const [selected, setSelected] = useState<Set<string>>(new Set(initial.allowed_channel_ids));
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  const toggle = (id: string) => {
    setSaved(false);
    setSelected((cur) => {
      const next = new Set(cur);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const onSave = async () => {
    setError(null);
    setSaved(false);
    try {
      await save.mutateAsync({ restricted, channel_ids: restricted ? [...selected] : [] });
      setSaved(true);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <div className="border-separator bg-elevated/40 mt-2 flex flex-col gap-2 rounded-lg border p-3">
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={restricted}
          onChange={(e) => {
            setRestricted(e.target.checked);
            setSaved(false);
          }}
        />
        <span className="text-fg">Restrict to selected channels</span>
      </label>
      {!restricted && (
        <p className="text-fg-subtle text-xs">This member can publish to every channel.</p>
      )}
      {restricted && (
        <div className="flex flex-col gap-1.5 pl-6">
          {channels?.length === 0 && (
            <p className="text-fg-subtle text-xs">No channels connected yet.</p>
          )}
          {channels?.map((c) => (
            <label key={c.id} className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={selected.has(c.id)} onChange={() => toggle(c.id)} />
              <span className="text-fg">{atHandle(c.handle)}</span>
              <span className="text-fg-subtle text-xs">{c.platform}</span>
            </label>
          ))}
        </div>
      )}
      <div className="flex items-center gap-3">
        <Button size="sm" onClick={onSave} disabled={save.isPending}>
          {save.isPending ? "Saving…" : "Save access"}
        </Button>
        {saved && <span className="text-success text-xs">Saved.</span>}
        {error && <span className="text-danger text-xs">{error}</span>}
      </div>
    </div>
  );
}
