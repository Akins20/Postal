"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { Film, ImagePlus, X } from "lucide-react";
import { useState } from "react";

import { mediaDownloadURL, useMedia, type Asset } from "@/data/media";
import type { MediaMeta } from "@/data/posts";
import { formatBytes } from "@/lib/format";
import { Button } from "@/ui/primitives/button";
import { EmptyState } from "@/ui/primitives/empty-state";
import { Icon } from "@/ui/primitives/icon";
import { Spinner } from "@/ui/primitives/spinner";

function toMeta(asset: Asset): MediaMeta {
  return {
    media_id: asset.id,
    kind: asset.kind as MediaMeta["kind"],
    mime: asset.mime,
    bytes: asset.bytes,
  };
}

/**
 * Attached-media chips plus an "Attach media" picker over the workspace
 * library (upload new files on the Media page).
 */
export function MediaAttach({
  workspaceId,
  attached,
  onChange,
}: {
  workspaceId: string;
  attached: MediaMeta[];
  onChange: (media: MediaMeta[]) => void;
}) {
  const [open, setOpen] = useState(false);
  const { data: assets, isPending } = useMedia(open ? workspaceId : undefined);

  const pick = (asset: Asset) => {
    if (!attached.some((m) => m.media_id === asset.id)) {
      onChange([...attached, toMeta(asset)]);
    }
    setOpen(false);
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        {attached.map((m) => (
          <span
            key={m.media_id}
            className="border-separator bg-elevated text-fg-muted flex items-center gap-1.5 rounded-full border py-1 pr-1 pl-3 text-xs"
          >
            {m.kind} · {formatBytes(m.bytes)}
            <button
              type="button"
              aria-label={`Remove attached ${m.kind}`}
              onClick={() => onChange(attached.filter((a) => a.media_id !== m.media_id))}
              className="hover:bg-fg/8 focus-visible:ring-ring inline-flex h-5 w-5 items-center justify-center rounded-full focus-visible:ring-2 focus-visible:outline-none"
            >
              <Icon icon={X} size={12} />
            </button>
          </span>
        ))}
        <Dialog.Root open={open} onOpenChange={setOpen}>
          <Dialog.Trigger asChild>
            <Button variant="secondary" size="sm">
              <Icon icon={ImagePlus} size={15} />
              Attach media
            </Button>
          </Dialog.Trigger>
          <Dialog.Portal>
            <Dialog.Overlay className="fixed inset-0 z-50 bg-black/30" />
            <Dialog.Content className="material-panel shadow-window fixed top-1/2 left-1/2 z-50 max-h-[80vh] w-[calc(100vw-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-auto rounded-xl p-6 outline-none">
              <Dialog.Title className="text-fg text-base font-semibold">Attach media</Dialog.Title>
              <Dialog.Description className="text-fg-muted mt-1 mb-4 text-sm">
                Pick from this workspace&apos;s library. Upload new files on the Media page.
              </Dialog.Description>
              {isPending && (
                <div className="py-8 text-center">
                  <Spinner label="Loading media" />
                </div>
              )}
              {assets?.length === 0 && (
                <EmptyState
                  title="The library is empty"
                  description="Upload images, GIFs or videos on the Media page first."
                  className="py-8"
                />
              )}
              {assets && assets.length > 0 && (
                <ul className="grid list-none grid-cols-3 gap-2">
                  {assets.map((a) => (
                    <li key={a.id}>
                      <button
                        type="button"
                        onClick={() => pick(a)}
                        className="border-separator bg-elevated hover:border-accent focus-visible:ring-ring block w-full overflow-hidden rounded-lg border focus-visible:ring-2 focus-visible:outline-none"
                      >
                        <span className="bg-fg/5 flex aspect-square items-center justify-center overflow-hidden">
                          {a.kind === "video" ? (
                            <Icon icon={Film} size={24} className="text-fg-muted" label="Video" />
                          ) : (
                            // eslint-disable-next-line @next/next/no-img-element -- API-served, auth'd blob
                            <img
                              src={mediaDownloadURL(workspaceId, a.id)}
                              alt={`${a.kind} (${a.mime})`}
                              className="h-full w-full object-cover"
                              loading="lazy"
                            />
                          )}
                        </span>
                        <span className="text-fg-subtle block truncate px-2 py-1 text-[11px]">
                          {a.mime} · {formatBytes(a.bytes)}
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
              <Dialog.Close asChild>
                <button
                  type="button"
                  aria-label="Close media picker"
                  className="text-fg hover:bg-fg/8 focus-visible:ring-ring absolute top-3 right-3 inline-flex h-8 w-8 items-center justify-center rounded-md focus-visible:ring-2 focus-visible:outline-none"
                >
                  <Icon icon={X} size={16} />
                </button>
              </Dialog.Close>
            </Dialog.Content>
          </Dialog.Portal>
        </Dialog.Root>
      </div>
    </div>
  );
}
