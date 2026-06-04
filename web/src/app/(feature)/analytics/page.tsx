import { BarChart3 } from "lucide-react";

import { EmptyState } from "@/ui/primitives/empty-state";

export default function AnalyticsPage() {
  return (
    <EmptyState
      icon={BarChart3}
      title="Analytics"
      description="Track post performance across channels. Arrives in sub-phase 12.6."
    />
  );
}
