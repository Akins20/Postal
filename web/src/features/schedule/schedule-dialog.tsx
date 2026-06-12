"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import Link from "next/link";
import { useState, type ReactNode } from "react";

import { useSchedulePost } from "@/data/schedule";
import type { NormalizedError } from "@/lib/api-error";
import { Button } from "@/ui/primitives/button";
import { Hint } from "@/ui/primitives/hint";
import { Icon } from "@/ui/primitives/icon";

type Mode = "slots" | "time";

/**
 * Schedule a saved draft: either into each channel's next open posting slot,
 * or at one specific time (picked in the user's local timezone and sent UTC).
 */
export function ScheduleDialog({
  workspaceId,
  postId,
  trigger,
}: {
  workspaceId: string;
  postId: string;
  trigger: ReactNode;
}) {
  const schedule = useSchedulePost(workspaceId);
  const [open, setOpen] = useState(false);
  const [mode, setMode] = useState<Mode>("slots");
  const [when, setWhen] = useState("");
  const [error, setError] = useState<NormalizedError | null>(null);
  const [scheduled, setScheduled] = useState<number | null>(null);

  const reset = () => {
    setMode("slots");
    setWhen("");
    setError(null);
    setScheduled(null);
  };

  const submit = async () => {
    setError(null);
    try {
      const jobs = await schedule.mutateAsync(
        mode === "slots"
          ? { postId, toSlots: true }
          : { postId, runAt: new Date(when).toISOString() },
      );
      setScheduled(jobs.length);
    } catch (e) {
      setError(e as NormalizedError);
    }
  };

  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) reset();
      }}
    >
      <Dialog.Trigger asChild>{trigger}</Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/45 backdrop-blur-[2px]" />
        <Dialog.Content className="material-dialog shadow-popover fixed top-1/2 left-1/2 z-50 w-[calc(100vw-2rem)] max-w-sm -translate-x-1/2 -translate-y-1/2 rounded-xl p-6 outline-none">
          <Dialog.Title className="text-fg text-base font-semibold">Schedule post</Dialog.Title>
          <Dialog.Description className="text-fg-muted mt-1 mb-4 text-sm">
            One job is created per selected channel.
          </Dialog.Description>

          {scheduled !== null ? (
            <div role="status" className="flex flex-col gap-4">
              <p className="text-fg text-sm">
                Scheduled - {scheduled} job{scheduled === 1 ? "" : "s"} created. Track them on the
                calendar.
              </p>
              <Dialog.Close asChild>
                <Button>Done</Button>
              </Dialog.Close>
            </div>
          ) : (
            <fieldset className="flex flex-col gap-3">
              <legend className="sr-only">When to publish</legend>
              <label className="flex items-start gap-2.5 text-sm">
                <input
                  type="radio"
                  name="schedule-mode"
                  checked={mode === "slots"}
                  onChange={() => setMode("slots")}
                  className="mt-0.5"
                />
                <span>
                  <span className="text-fg flex items-center gap-1.5 font-medium">
                    Next open slots
                    <Hint label="About posting slots">
                      Each channel keeps a weekly posting schedule (its slots). This drops every
                      variant into its channel&apos;s earliest free slot. Manage slots on the
                      Calendar page.
                    </Hint>
                  </span>
                  <span className="text-fg-muted block text-xs">
                    Use each channel&apos;s posting schedule.
                  </span>
                </span>
              </label>
              <label className="flex items-start gap-2.5 text-sm">
                <input
                  type="radio"
                  name="schedule-mode"
                  checked={mode === "time"}
                  onChange={() => setMode("time")}
                  className="mt-0.5"
                />
                <span className="flex-1">
                  <span className="text-fg block font-medium">Specific time</span>
                  <span className="text-fg-muted block text-xs">
                    In your timezone ({Intl.DateTimeFormat().resolvedOptions().timeZone}).
                  </span>
                  {mode === "time" && (
                    <input
                      type="datetime-local"
                      aria-label="Publish at"
                      value={when}
                      onChange={(e) => setWhen(e.target.value)}
                      className="border-separator bg-elevated text-fg focus-visible:ring-ring mt-2 h-9 w-full rounded-md border px-2 text-sm focus-visible:ring-2 focus-visible:outline-none"
                    />
                  )}
                </span>
              </label>

              {error && (
                <p role="alert" className="text-danger text-xs">
                  {error.message}
                  {error.code === "insufficient_credits" && (
                    <>
                      {" "}
                      <Link href="/wallet" className="text-accent font-medium hover:underline">
                        Open Wallet
                      </Link>
                    </>
                  )}
                </p>
              )}

              <div className="mt-2 flex justify-end gap-2">
                <Dialog.Close asChild>
                  <Button variant="secondary" disabled={schedule.isPending}>
                    Cancel
                  </Button>
                </Dialog.Close>
                <Button
                  onClick={submit}
                  disabled={schedule.isPending || (mode === "time" && !when)}
                >
                  {schedule.isPending ? "Scheduling…" : "Schedule"}
                </Button>
              </div>
            </fieldset>
          )}

          <Dialog.Close asChild>
            <button
              type="button"
              aria-label="Close"
              className="text-fg hover:bg-fg/8 focus-visible:ring-ring absolute top-3 right-3 inline-flex h-8 w-8 items-center justify-center rounded-md focus-visible:ring-2 focus-visible:outline-none"
            >
              <Icon icon={X} size={16} />
            </button>
          </Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
