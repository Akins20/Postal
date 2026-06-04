import { Settings } from "lucide-react";

import { EmptyState } from "@/ui/primitives/empty-state";

export default function SettingsPage() {
  return (
    <EmptyState
      icon={Settings}
      title="Settings"
      description="Account and workspace settings. Arrives in sub-phase 12.7."
    />
  );
}
