"use client";

import { useState } from "react";

import { usePosts, useDeletePost, type Post } from "@/data/posts";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { ConfirmDialog } from "@/ui/primitives/confirm-dialog";
import { Spinner } from "@/ui/primitives/spinner";
import { StatusPill } from "@/ui/primitives/status-pill";

function DraftRow({
  workspaceId,
  post,
  onEdit,
}: {
  workspaceId: string;
  post: Post;
  onEdit: (post: Post) => void;
}) {
  const remove = useDeletePost(workspaceId);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const excerpt = post.variants?.[0]?.body ?? "";
  const channelCount = post.variants?.length ?? 0;

  const onDelete = async () => {
    setError(null);
    try {
      await remove.mutateAsync({ postId: post.id });
      setOpen(false);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <li className="border-separator flex flex-wrap items-center gap-3 border-b py-3 last:border-0">
      <div className="min-w-0 flex-1">
        <p className="text-fg truncate text-sm">{excerpt || <em>(no text)</em>}</p>
        <p className="text-fg-subtle text-xs">
          {channelCount} channel{channelCount === 1 ? "" : "s"} ·{" "}
          {new Date(post.created_at).toLocaleDateString()}
        </p>
      </div>
      <StatusPill tone={post.status === "draft" ? "neutral" : "accent"}>{post.status}</StatusPill>
      <Button variant="secondary" size="sm" onClick={() => onEdit(post)}>
        Edit
      </Button>
      <ConfirmDialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (!next) setError(null);
        }}
        trigger={
          <Button variant="ghost" size="sm">
            Delete
          </Button>
        }
        title="Delete this draft?"
        description={
          <>
            The draft and its per-channel variants are removed. Scheduled jobs for it are not.
            {error && (
              <span role="alert" className="text-danger mt-2 block">
                {error}
              </span>
            )}
          </>
        }
        confirmLabel="Delete"
        destructive
        pending={remove.isPending}
        onConfirm={onDelete}
      />
    </li>
  );
}

/** Existing posts in the workspace; Edit loads one back into the composer. */
export function DraftsList({
  workspaceId,
  onEdit,
}: {
  workspaceId: string;
  onEdit: (post: Post) => void;
}) {
  const { data: posts, isPending, isError } = usePosts(workspaceId);

  if (isPending) {
    return (
      <div className="py-6 text-center">
        <Spinner label="Loading posts" />
      </div>
    );
  }
  if (isError) {
    return (
      <p role="alert" className="text-danger text-sm">
        Couldn&apos;t load posts. Please try again.
      </p>
    );
  }
  if (!posts || posts.length === 0) {
    return <p className="text-fg-muted py-2 text-sm">Nothing saved yet — your drafts land here.</p>;
  }
  return (
    <ul className="flex list-none flex-col">
      {posts.map((p) => (
        <DraftRow key={p.id} workspaceId={workspaceId} post={p} onEdit={onEdit} />
      ))}
    </ul>
  );
}
