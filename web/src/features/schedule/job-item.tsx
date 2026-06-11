"use client";

import { format } from "date-fns";
import { useState } from "react";

import type { Channel } from "@/data/channels";
import { useCancelJob, type Job, type JobStatus } from "@/data/schedule";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { ConfirmDialog } from "@/ui/primitives/confirm-dialog";
import { StatusPill } from "@/ui/primitives/status-pill";

export const JOB_TONE: Record<JobStatus, "neutral" | "accent" | "success" | "warning" | "danger"> =
  {
    scheduled: "accent",
    publishing: "warning",
    published: "success",
    failed: "danger",
    canceled: "neutral",
  };

/** One scheduled job row (week view / day detail): time, channel, status, cancel. */
export function JobItem({
  workspaceId,
  job,
  channel,
}: {
  workspaceId: string;
  job: Job;
  channel?: Channel;
}) {
  const cancel = useCancelJob(workspaceId);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onCancel = async () => {
    setError(null);
    try {
      await cancel.mutateAsync({ jobId: job.id });
      setOpen(false);
    } catch (e) {
      setError((e as NormalizedError).message);
    }
  };

  return (
    <div className="border-separator flex flex-wrap items-center gap-2 border-b py-2 text-sm last:border-0">
      <span className="text-fg w-16 font-medium tabular-nums">
        {format(new Date(job.run_at), "HH:mm")}
      </span>
      <span className="text-fg-muted min-w-0 flex-1 truncate">
        {channel ? `@${channel.handle}` : job.channel_id.slice(0, 8)}
        {job.status === "failed" && job.last_error ? ` — ${job.last_error}` : ""}
      </span>
      <StatusPill tone={JOB_TONE[job.status]}>{job.status}</StatusPill>
      {job.status === "scheduled" && (
        <ConfirmDialog
          open={open}
          onOpenChange={(next) => {
            setOpen(next);
            if (!next) setError(null);
          }}
          trigger={
            <Button variant="ghost" size="sm">
              Cancel
            </Button>
          }
          title="Cancel this scheduled post?"
          description={
            <>
              It won&apos;t be published to {channel ? `@${channel.handle}` : "its channel"}. The
              draft itself is kept.
              {error && (
                <span role="alert" className="text-danger mt-2 block">
                  {error}
                </span>
              )}
            </>
          }
          confirmLabel="Cancel job"
          destructive
          pending={cancel.isPending}
          onConfirm={onCancel}
        />
      )}
    </div>
  );
}
