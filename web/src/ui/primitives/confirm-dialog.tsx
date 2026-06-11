"use client";

import * as Dialog from "@radix-ui/react-dialog";
import type { ReactNode } from "react";

import { Button } from "./button";

/**
 * A confirmation dialog for destructive or hard-to-undo actions. The trigger
 * element is passed as `trigger`; confirm runs `onConfirm` and closes unless
 * the action is pending.
 */
export function ConfirmDialog({
  trigger,
  title,
  description,
  confirmLabel = "Confirm",
  destructive = false,
  pending = false,
  open,
  onOpenChange,
  onConfirm,
}: {
  trigger: ReactNode;
  title: string;
  description: ReactNode;
  confirmLabel?: string;
  destructive?: boolean;
  pending?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Trigger asChild>{trigger}</Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/30" />
        <Dialog.Content className="material-panel shadow-window fixed top-1/2 left-1/2 z-50 w-[calc(100vw-2rem)] max-w-sm -translate-x-1/2 -translate-y-1/2 rounded-xl p-6 outline-none">
          <Dialog.Title className="text-fg text-base font-semibold">{title}</Dialog.Title>
          <Dialog.Description className="text-fg-muted mt-2 text-sm">
            {description}
          </Dialog.Description>
          <div className="mt-5 flex justify-end gap-2">
            <Dialog.Close asChild>
              <Button variant="secondary" disabled={pending}>
                Cancel
              </Button>
            </Dialog.Close>
            <Button
              variant={destructive ? "danger" : "primary"}
              disabled={pending}
              onClick={onConfirm}
            >
              {pending ? "Working…" : confirmLabel}
            </Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
