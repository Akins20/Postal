"use client";

import { Upload } from "lucide-react";
import { useRef, useState } from "react";

import { useUploadMedia } from "@/data/media";
import { Button } from "@/ui/primitives/button";
import { Icon } from "@/ui/primitives/icon";

/**
 * Picks a file and uploads it with a live progress bar. Size caps (image 5 MiB,
 * GIF 15 MiB, video 512 MiB) and the workspace storage quota are enforced
 * server-side; rejections surface inline.
 */
export function UploadButton({ workspaceId }: { workspaceId: string }) {
  const upload = useUploadMedia(workspaceId);
  const inputRef = useRef<HTMLInputElement>(null);
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const onPick = async (file: File | undefined) => {
    if (!file) return;
    setError(null);
    setProgress(0);
    try {
      await upload.mutateAsync({ file, onProgress: setProgress });
    } catch (e) {
      setError((e as { message: string }).message);
    } finally {
      setProgress(null);
      if (inputRef.current) inputRef.current.value = "";
    }
  };

  return (
    <div className="flex flex-col items-end gap-1.5">
      <input
        ref={inputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/gif,video/mp4,video/quicktime"
        className="sr-only"
        aria-label="Choose a file to upload"
        onChange={(e) => onPick(e.target.files?.[0])}
      />
      <Button onClick={() => inputRef.current?.click()} disabled={upload.isPending}>
        <Icon icon={Upload} size={16} />
        {upload.isPending ? "Uploading…" : "Upload"}
      </Button>
      {progress !== null && (
        <progress
          value={Math.round(progress * 100)}
          max={100}
          aria-label="Upload progress"
          className="h-1.5 w-36 overflow-hidden rounded-full"
        />
      )}
      {error && (
        <p role="alert" className="text-danger text-xs">
          {error}
        </p>
      )}
    </div>
  );
}
