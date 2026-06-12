"use client";

// Client component: destination cards carry icon COMPONENTS (see config/nav).
import Link from "next/link";

import { dockItems, dockManage } from "@/config/nav";
import { OverviewWidgets } from "@/features/dashboard/overview-widgets";
import { Hint } from "@/ui/primitives/hint";
import { Icon } from "@/ui/primitives/icon";
import { Panel } from "@/ui/primitives/panel";

/** Dashboard home: a welcome, the destination launchpad, and live widgets. */
export default function DashboardPage() {
  const destinations = [...dockItems.filter((d) => d.href !== "/"), ...dockManage];

  return (
    <div className="h-full overflow-auto">
      <main id="main" className="mx-auto flex max-w-5xl flex-col gap-6 px-6 py-6 pb-32">
        <Panel className="flex items-start justify-between gap-4 p-6">
          <div className="flex flex-col gap-1">
            <h1 className="text-fg text-xl font-semibold tracking-tight">Welcome to Postal</h1>
            <p className="text-fg-muted max-w-prose text-sm">
              Compose once, publish everywhere, and schedule it all. Free, no paywall. Pick a
              destination below, or use the dock at the bottom.
            </p>
          </div>
          <Hint>
            The dock is your primary navigation. Each section opens a focused workspace with its own
            sidebar.
          </Hint>
        </Panel>

        <section
          aria-label="Destinations"
          className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3"
        >
          {destinations.map((d) => (
            <Link
              key={d.href}
              href={d.href}
              className="group rounded-xl focus-visible:outline-none"
            >
              <Panel className="group-hover:bg-elevated group-focus-visible:ring-ring flex h-full items-center gap-3 p-4 transition-all group-hover:-translate-y-0.5 group-focus-visible:ring-2">
                <div className="bg-accent/12 text-accent flex h-10 w-10 items-center justify-center rounded-lg">
                  <Icon icon={d.icon} size={20} />
                </div>
                <div className="flex flex-col">
                  <span className="text-fg text-sm font-medium">{d.label}</span>
                  <span className="text-fg-muted text-xs">Open {d.label.toLowerCase()}</span>
                </div>
              </Panel>
            </Link>
          ))}
        </section>

        <OverviewWidgets />
      </main>
    </div>
  );
}
