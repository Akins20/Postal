import { featureSidebar } from "@/config/nav";
import { FeatureShell } from "@/ui/sidebar/feature-shell";

/**
 * Layout for feature/sub-routes — wraps them in the macOS side-rail shell. The
 * dashboard (app/page.tsx) sits outside this group and uses the Dock instead.
 */
export default function FeatureLayout({ children }: { children: React.ReactNode }) {
  return (
    <FeatureShell title="Postal" sections={featureSidebar}>
      {children}
    </FeatureShell>
  );
}
