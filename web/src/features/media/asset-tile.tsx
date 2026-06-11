"use client";

import { Film, Trash2 } from "lucide-react";
import { useState } from "react";

import { mediaDownloadURL, useDeleteMedia, type Asset } from "@/data/media";
import type { NormalizedError } from "@/lib/api-error";
import { formatBytes } from "@/lib/format";
import { Button } from "@/ui/primitives/button";
import { ConfirmDialog } from "@/ui/primitives/confirm-dialog";
import { Icon } from "@/ui/primitives/icon";

/** One asset in the media grid: preview, meta, delete. */
export function AssetTile({ workspaceId, asset }: { workspaceId: string; asset: Asset }) {
  const remove = useDeleteMedia(workspaceId);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const isVideo = asset.kind === "video";

  const onDelete = async () => {
    setError(null);
    try {
      await remove.mutateAsync({ mediaId: asset.id });
      setOpen(false);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <figure className="border-separator bg-elevated group relative flex flex-col overflow-hidden rounded-lg border">
      <div className="bg-fg/5 flex aspect-square items-center justify-center overflow-hidden">
        {isVideo ? (
          <Icon icon={Film} size={28} className="text-fg-muted" label="Video" />
        ) : (
          // Cookie-authenticated bytes endpoint; alt text from kind + mime.
          // eslint-disable-next-line @next/next/no-img-element -- API-served, auth'd blob; next/image can't proxy it
          <img
            src={mediaDownloadURL(workspaceId, asset.id)}
            alt={`${asset.kind} asset (${asset.mime})`}
            className="h-full w-full object-cover"
            loading="lazy"
          />
        )}
      </div>
      <figcaption className="flex items-center justify-between gap-2 px-2.5 py-2">
        <span className="text-fg-muted truncate text-xs">
          {asset.mime} · {formatBytes(asset.bytes)}
        </span>
        <ConfirmDialog
          open={open}
          onOpenChange={(next) => {
            setOpen(next);
            if (!next) setError(null);
          }}
          trigger={
            <Button variant="ghost" size="sm" aria-label={`Delete ${asset.kind} asset`}>
              <Icon icon={Trash2} size={15} />
            </Button>
          }
          title="Delete this asset?"
          description={
            <>
              Posts already referencing it keep their copy, but it can&apos;t be attached to new
              posts.
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
      </figcaption>
    </figure>
  );
}
