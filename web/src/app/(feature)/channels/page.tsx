import { Radio } from "lucide-react";

import { EmptyState } from "@/ui/primitives/empty-state";

export default function ChannelsPage() {
  return (
    <EmptyState
      icon={Radio}
      title="Channels"
      description="Connect your social accounts. Arrives in sub-phase 12.3."
    />
  );
}
