"use client";

// Client component: the dock nav config carries icon COMPONENTS, which a
// Server Component can't pass across the RSC boundary (not serializable).
import { dockItems, dockManage } from "@/config/nav";
import { AuthGuard } from "@/features/auth/auth-guard";
import { AppHeader } from "@/features/shell/app-header";
import { Dock } from "@/ui/dock/dock";

/**
 * Authenticated app shell: everything under (app) requires a signed-in user
 * and shares one chrome - the global header on top and the macOS dock at the
 * bottom of EVERY page (the dashboard is no longer special). Feature routes
 * add their side rail inside this frame.
 */
export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard>
      <div className="flex h-dvh flex-col">
        <AppHeader />
        <div className="min-h-0 flex-1">{children}</div>
        <Dock groups={[dockItems, dockManage]} />
      </div>
    </AuthGuard>
  );
}
