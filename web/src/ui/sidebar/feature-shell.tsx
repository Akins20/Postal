"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { Menu, X } from "lucide-react";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { Icon } from "@/ui/primitives/icon";

import { Sidebar, type SidebarSection } from "./sidebar";

/**
 * Feature-route chrome (FRONTEND_PLAN §5): a persistent macOS side rail on
 * tablet/desktop, and a slide-over sheet behind a menu button on mobile. Closes
 * the sheet on navigation. The dashboard uses the Dock instead of this shell.
 */
export function FeatureShell({
  title,
  sections,
  header,
  children,
}: {
  title: string;
  sections: SidebarSection[];
  header?: ReactNode;
  children: ReactNode;
}) {
  const pathname = usePathname();

  return (
    <div className="flex h-dvh flex-col md:flex-row">
      {/* Tablet/desktop: persistent sidebar */}
      <div className="hidden md:block">
        <Sidebar sections={sections} header={header} />
      </div>

      {/* Mobile: top bar + slide-over sheet. Keying by pathname remounts (and
          closes) the sheet on navigation without a setState-in-effect. */}
      <Dialog.Root key={pathname}>
        <header className="border-separator flex items-center gap-2 border-b px-3 py-2 md:hidden">
          <Dialog.Trigger asChild>
            <button
              type="button"
              aria-label="Open navigation"
              className="text-fg hover:bg-fg/8 focus-visible:ring-ring inline-flex h-9 w-9 items-center justify-center rounded-md focus-visible:ring-2 focus-visible:outline-none"
            >
              <Icon icon={Menu} size={20} />
            </button>
          </Dialog.Trigger>
          <span className="text-fg text-sm font-semibold">{title}</span>
        </header>

        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 z-50 bg-black/45 backdrop-blur-[2px] md:hidden" />
          <Dialog.Content className="fixed inset-y-0 left-0 z-50 outline-none md:hidden">
            <Dialog.Title className="sr-only">{title} navigation</Dialog.Title>
            <Sidebar sections={sections} header={header} className="h-full" />
            <Dialog.Close asChild>
              <button
                type="button"
                aria-label="Close navigation"
                className="text-fg hover:bg-fg/8 focus-visible:ring-ring absolute top-2 right-2 inline-flex h-8 w-8 items-center justify-center rounded-md focus-visible:ring-2 focus-visible:outline-none"
              >
                <Icon icon={X} size={18} />
              </button>
            </Dialog.Close>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      <main id="main" className="flex-1 overflow-auto">
        {children}
      </main>
    </div>
  );
}
