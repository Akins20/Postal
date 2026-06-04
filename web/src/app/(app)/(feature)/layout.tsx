import { WorkspaceSwitcher } from "@/features/workspace/workspace-switcher";

import { featureSidebar } from "@/config/nav";
import { FeatureShell } from "@/ui/sidebar/feature-shell";

/**
 * Layout for feature/sub-routes — wraps them in the macOS side-rail shell with
 * the workspace switcher in the rail header. The dashboard sits outside this
 * group and uses the Dock instead.
 */
export default function FeatureLayout({ children }: { children: React.ReactNode }) {
  return (
    <FeatureShell
      title="Postal"
      sections={featureSidebar}
      header={
        <div className="px-1 pb-1">
          <WorkspaceSwitcher />
        </div>
      }
    >
      {children}
    </FeatureShell>
  );
}
