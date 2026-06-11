"use client";

import { ImageIcon } from "lucide-react";

import { useMedia } from "@/data/media";
import { useActiveWorkspace } from "@/features/workspace/use-active-workspace";
import { EmptyState } from "@/ui/primitives/empty-state";
import { Hint } from "@/ui/primitives/hint";
import { Panel } from "@/ui/primitives/panel";
import { Spinner } from "@/ui/primitives/spinner";

import { AssetTile } from "./asset-tile";
import { UploadButton } from "./upload-button";

/** The Media screen: workspace asset library with upload + delete. */
export function MediaPanel() {
  const { active } = useActiveWorkspace();
  const { data: assets, isPending, isError } = useMedia(active?.id);

  if (!active) {
    return (
      <div className="py-10 text-center">
        <Spinner label="Loading workspace" />
      </div>
    );
  }

  return (
    <Panel className="p-6">
      <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-1.5">
            <h2 className="text-fg text-sm font-semibold">Library</h2>
            <Hint label="About the media library">
              Images, GIFs and videos you upload here can be attached to posts in the composer.
              Caps: image 5 MiB, GIF 15 MiB, video 512 MiB, plus a per-workspace storage quota.
            </Hint>
          </div>
          <p className="text-fg-muted mt-1 text-sm">Everything this workspace has uploaded.</p>
        </div>
        <UploadButton workspaceId={active.id} />
      </div>

      {isPending && (
        <div className="py-6 text-center">
          <Spinner label="Loading media" />
        </div>
      )}
      {isError && (
        <p role="alert" className="text-danger text-sm">
          Couldn&apos;t load media. Please try again.
        </p>
      )}
      {assets?.length === 0 && (
        <EmptyState
          icon={ImageIcon}
          title="No media yet"
          description="Upload an image, GIF or video to attach it to your posts."
          className="py-10"
        />
      )}
      {assets && assets.length > 0 && (
        <ul className="grid list-none grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
          {assets.map((a) => (
            <li key={a.id}>
              <AssetTile workspaceId={active.id} asset={a} />
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
