"use client";

import { Radio } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { useChannels } from "@/data/channels";
import { usePost } from "@/data/posts";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { Button } from "@/ui/primitives/button";
import { EmptyState } from "@/ui/primitives/empty-state";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";

import { Composer } from "./composer";
import { DraftsList } from "./drafts-list";

/** The Compose screen: the compose-once editor plus saved drafts. */
export function ComposeScreen() {
  const { active } = useActiveWorkspace();
  const { data: channels, isPending } = useChannels(active?.id);
  // Editing loads the post DETAIL - the list endpoint omits variants, so the
  // list row alone can't seed the editor.
  const [editingId, setEditingId] = useState<string | null>(null);
  const detail = usePost(active?.id, editingId ?? undefined);

  if (!active || isPending) {
    return (
      <div className="py-10 text-center">
        <Spinner label="Loading composer" />
      </div>
    );
  }

  if (!channels || channels.length === 0) {
    return (
      <Panel>
        <EmptyState
          icon={Radio}
          title="Connect a channel first"
          description="Posts are written once and published to the channels you pick - connect a social account to start composing."
          action={
            <Button asChild>
              <Link href="/channels">Go to Channels</Link>
            </Button>
          }
        />
      </Panel>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {editingId && (
        <Panel className="flex items-center justify-between gap-3 p-4">
          <p className="text-fg-muted text-sm">Editing a saved draft.</p>
          <Button variant="ghost" size="sm" onClick={() => setEditingId(null)}>
            New post
          </Button>
        </Panel>
      )}
      {editingId && detail.isPending ? (
        <div className="py-10 text-center">
          <Spinner label="Loading draft" />
        </div>
      ) : (
        <Composer
          key={editingId ?? "new"}
          workspaceId={active.id}
          channels={channels}
          initial={editingId ? detail.data : undefined}
        />
      )}

      <Panel className="p-6">
        <h2 className="text-fg mb-2 text-sm font-semibold">Your posts</h2>
        <DraftsList workspaceId={active.id} onEdit={(post) => setEditingId(post.id)} />
      </Panel>
    </div>
  );
}
